package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type seedState struct {
	TenantID string `json:"tenantId"`
	UserID   string `json:"userId"`
	SiteID   string `json:"siteId"`
}

func createDatabaseIfMissing(cfg Config) error {
	admin, err := sql.Open("pgx", cfg.adminDSN())
	if err != nil {
		return err
	}
	defer admin.Close()
	if err := admin.Ping(); err != nil {
		return err
	}
	var exists bool
	if err := admin.QueryRow(`select exists(select 1 from pg_database where datname = $1)`, cfg.DBName).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = admin.Exec(`create database ` + quoteIdentifier(cfg.DBName))
	return err
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func migrate(db *sql.DB) error {
	if err := detectSchemaDrift(db); err != nil {
		return err
	}
	steps := []struct {
		version string
		sql     string
	}{
		{"20260425_001_core", `
create table if not exists schema_migrations (
	version text primary key,
	applied_at timestamptz not null default now()
);

create table if not exists tenants (
	id uuid primary key,
	name text not null,
	slug text not null unique,
	status text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists users (
	id uuid primary key,
	email text not null unique,
	name text not null,
	password_hash text,
	status text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists tenant_users (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	user_id uuid not null references users(id) on delete cascade,
	role text not null,
	status text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (tenant_id, user_id)
);

create table if not exists sites (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	name text not null,
	slug text not null,
	status text not null,
	active_release_id uuid,
	published_at timestamptz,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (tenant_id, slug)
);

create table if not exists workspace_files (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	path text not null,
	file_type text not null,
	size_bytes bigint not null default 0,
	checksum text,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (site_id, path)
);

create table if not exists site_releases (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	release_number integer not null,
	artifact_path text not null,
	public_url text,
	status text not null,
	created_at timestamptz not null default now(),
	unique (site_id, release_number)
);

create table if not exists assets (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	path text not null,
	file_name text not null,
	content_type text not null,
	size_bytes bigint not null default 0,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists audit_logs (
	id uuid primary key,
	tenant_id uuid references tenants(id) on delete set null,
	site_id uuid references sites(id) on delete set null,
	actor_id uuid references users(id) on delete set null,
	action text not null,
	target_type text not null,
	target_id text,
	outcome text not null,
	metadata_json jsonb,
	created_at timestamptz not null default now()
);

create index if not exists idx_sites_tenant_id on sites(tenant_id);
create index if not exists idx_audit_logs_tenant_id on audit_logs(tenant_id);
`},
		{"20260425_002_modules", `
create table if not exists products (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	name text not null,
	slug text not null,
	description text not null default '',
	price numeric(12,2) not null default 0,
	sku text not null default '',
	image_url text not null default '',
	category text not null default '',
	stock_status text not null default 'in_stock',
	status text not null default 'draft',
	seo_title text not null default '',
	seo_description text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (site_id, slug)
);
create table if not exists orders (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	order_number text not null,
	customer_name text not null,
	customer_email text not null default '',
	amount numeric(12,2) not null default 0,
	payment_status text not null default 'pending',
	fulfillment_status text not null default 'unfulfilled',
	status text not null default 'open',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create table if not exists customers (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	name text not null,
	email text not null,
	orders_count integer not null default 0,
	total_spent numeric(12,2) not null default 0,
	last_order_at timestamptz,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create table if not exists coupons (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	code text not null,
	discount_type text not null,
	value numeric(12,2) not null default 0,
	expires_at timestamptz,
	usage_limit integer not null default 0,
	usage_count integer not null default 0,
	status text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (site_id, code)
);
create table if not exists store_settings (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null unique references sites(id) on delete cascade,
	currency text not null default 'USD',
	store_email text not null default '',
	checkout_mode text not null default 'standard',
	payment_provider text not null default 'coming_soon',
	order_notifications_enabled boolean not null default true,
	settings_json jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists shipping_zones (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	name text not null,
	region text not null default '',
	method text not null default 'flat_rate',
	rate numeric(12,2) not null default 0,
	status text not null default 'active',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create table if not exists tax_settings (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null unique references sites(id) on delete cascade,
	taxes_enabled boolean not null default true,
	prices_include_tax boolean not null default false,
	default_region text not null default 'US',
	settings_json jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists blog_posts (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	title text not null,
	slug text not null,
	excerpt text not null default '',
	content text not null default '',
	featured_image_url text not null default '',
	author text not null default '',
	tags_json jsonb not null default '[]'::jsonb,
	seo_title text not null default '',
	seo_description text not null default '',
	visibility text not null default 'public',
	status text not null default 'draft',
	published_at timestamptz,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (site_id, slug)
);
create table if not exists blog_categories (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	name text not null,
	slug text not null,
	description text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (site_id, slug)
);
create table if not exists blog_tags (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	name text not null,
	slug text not null,
	visibility text not null default 'public',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (site_id, slug)
);
create table if not exists blog_authors (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	name text not null,
	email text not null,
	role text not null default 'author',
	bio text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create table if not exists newsletter_settings (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null unique references sites(id) on delete cascade,
	enabled boolean not null default true,
	sender_name text not null default '',
	sender_email text not null default '',
	send_posts_as_email boolean not null default true,
	settings_json jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create table if not exists blog_settings (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null unique references sites(id) on delete cascade,
	blog_title text not null default '',
	description text not null default '',
	default_author_id uuid,
	rss_enabled boolean not null default true,
	comments_enabled boolean not null default true,
	canonical_url text not null default '',
	posts_per_page integer not null default 10,
	social_sharing_enabled boolean not null default true,
	settings_json jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create table if not exists blog_comments (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	post_id uuid,
	author_name text not null default '',
	author_email text not null default '',
	content text not null default '',
	status text not null default 'pending',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists blog_members (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	name text not null,
	email text not null,
	member_type text not null default 'free',
	status text not null default 'active',
	joined_at timestamptz not null default now(),
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create table if not exists blog_analytics (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	post_id uuid,
	views integer not null default 0,
	signups integer not null default 0,
	newsletter_opens integer not null default 0,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create table if not exists subscription_plans (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	name text not null,
	description text not null default '',
	price numeric(12,2) not null default 0,
	billing_interval text not null default 'monthly',
	trial_days integer not null default 0,
	status text not null default 'draft',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create table if not exists subscription_subscribers (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	name text not null,
	email text not null,
	status text not null default 'active',
	joined_at timestamptz not null default now(),
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create table if not exists subscriptions (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	subscriber_id uuid references subscription_subscribers(id) on delete set null,
	plan_id uuid references subscription_plans(id) on delete set null,
	status text not null default 'active',
	renewal_date timestamptz,
	amount numeric(12,2) not null default 0,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create table if not exists subscription_invoices (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	invoice_number text not null,
	subscriber_id uuid references subscription_subscribers(id) on delete set null,
	amount numeric(12,2) not null default 0,
	status text not null default 'open',
	issued_at timestamptz,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (site_id, invoice_number)
);
create table if not exists failed_payments (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null references sites(id) on delete cascade,
	subscription_id uuid references subscriptions(id) on delete set null,
	subscriber_id uuid references subscription_subscribers(id) on delete set null,
	amount numeric(12,2) not null default 0,
	status text not null default 'pending',
	retry_count integer not null default 0,
	next_retry_at timestamptz,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
create table if not exists subscription_settings (
	id uuid primary key,
	tenant_id uuid not null references tenants(id) on delete cascade,
	site_id uuid not null unique references sites(id) on delete cascade,
	billing_currency text not null default 'USD',
	renewal_email_enabled boolean not null default true,
	cancellation_policy text not null default '',
	trial_settings_json jsonb not null default '{}'::jsonb,
	settings_json jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
`},
		{"20260425_003_reconcile", `
alter table if exists tenants add column if not exists status text not null default 'active';
alter table if exists tenants add column if not exists updated_at timestamptz not null default now();

alter table if exists users add column if not exists status text not null default 'active';
alter table if exists users add column if not exists updated_at timestamptz not null default now();
alter table if exists users add column if not exists password_hash text;

alter table if exists tenant_users add column if not exists status text not null default 'active';
alter table if exists tenant_users add column if not exists updated_at timestamptz not null default now();

alter table if exists sites add column if not exists active_release_id uuid;
alter table if exists sites add column if not exists published_at timestamptz;
alter table if exists sites add column if not exists updated_at timestamptz not null default now();
alter table if exists sites add column if not exists status text not null default 'draft';

alter table if exists workspace_files add column if not exists tenant_id uuid;
alter table if exists workspace_files add column if not exists file_type text not null default 'html';
alter table if exists workspace_files add column if not exists size_bytes bigint not null default 0;
alter table if exists workspace_files add column if not exists checksum text;
alter table if exists workspace_files add column if not exists updated_at timestamptz not null default now();

alter table if exists site_releases add column if not exists tenant_id uuid;
alter table if exists site_releases add column if not exists public_url text;
alter table if exists site_releases add column if not exists status text not null default 'published';

alter table if exists products add column if not exists tenant_id uuid;
alter table if exists products add column if not exists site_id uuid;
alter table if exists products add column if not exists sku text not null default '';
alter table if exists products add column if not exists image_url text not null default '';
alter table if exists products add column if not exists category text not null default '';
alter table if exists products add column if not exists stock_status text not null default 'in_stock';
alter table if exists products add column if not exists status text not null default 'draft';
alter table if exists products add column if not exists seo_title text not null default '';
alter table if exists products add column if not exists seo_description text not null default '';
alter table if exists products add column if not exists updated_at timestamptz not null default now();

alter table if exists orders add column if not exists tenant_id uuid;
alter table if exists orders add column if not exists site_id uuid;
alter table if exists orders add column if not exists payment_status text not null default 'pending';
alter table if exists orders add column if not exists fulfillment_status text not null default 'unfulfilled';
alter table if exists orders add column if not exists status text not null default 'open';
alter table if exists orders add column if not exists updated_at timestamptz not null default now();

alter table if exists customers add column if not exists tenant_id uuid;
alter table if exists customers add column if not exists site_id uuid;
alter table if exists customers add column if not exists orders_count integer not null default 0;
alter table if exists customers add column if not exists total_spent numeric(12,2) not null default 0;
alter table if exists customers add column if not exists last_order_at timestamptz;
alter table if exists customers add column if not exists updated_at timestamptz not null default now();

alter table if exists coupons add column if not exists tenant_id uuid;
alter table if exists coupons add column if not exists site_id uuid;
alter table if exists coupons add column if not exists usage_limit integer not null default 0;
alter table if exists coupons add column if not exists usage_count integer not null default 0;
alter table if exists coupons add column if not exists status text not null default 'active';
alter table if exists coupons add column if not exists updated_at timestamptz not null default now();

alter table if exists store_settings add column if not exists tenant_id uuid;
alter table if exists store_settings add column if not exists site_id uuid;
alter table if exists store_settings add column if not exists payment_provider text not null default 'coming_soon';
alter table if exists store_settings add column if not exists order_notifications_enabled boolean not null default true;
alter table if exists store_settings add column if not exists settings_json jsonb not null default '{}'::jsonb;
alter table if exists store_settings add column if not exists updated_at timestamptz not null default now();

alter table if exists shipping_zones add column if not exists tenant_id uuid;
alter table if exists shipping_zones add column if not exists site_id uuid;
alter table if exists shipping_zones add column if not exists rate numeric(12,2) not null default 0;
alter table if exists shipping_zones add column if not exists updated_at timestamptz not null default now();

alter table if exists tax_settings add column if not exists tenant_id uuid;
alter table if exists tax_settings add column if not exists site_id uuid;
alter table if exists tax_settings add column if not exists settings_json jsonb not null default '{}'::jsonb;
alter table if exists tax_settings add column if not exists updated_at timestamptz not null default now();

alter table if exists blog_posts add column if not exists tenant_id uuid;
alter table if exists blog_posts add column if not exists site_id uuid;
alter table if exists blog_posts add column if not exists featured_image_url text not null default '';
alter table if exists blog_posts add column if not exists author text not null default '';
alter table if exists blog_posts add column if not exists tags_json jsonb not null default '[]'::jsonb;
alter table if exists blog_posts add column if not exists seo_title text not null default '';
alter table if exists blog_posts add column if not exists seo_description text not null default '';
alter table if exists blog_posts add column if not exists visibility text not null default 'public';
alter table if exists blog_posts add column if not exists status text not null default 'draft';
alter table if exists blog_posts add column if not exists published_at timestamptz;
alter table if exists blog_posts add column if not exists updated_at timestamptz not null default now();

alter table if exists blog_categories add column if not exists tenant_id uuid;
alter table if exists blog_categories add column if not exists site_id uuid;
alter table if exists blog_categories add column if not exists description text not null default '';
alter table if exists blog_categories add column if not exists updated_at timestamptz not null default now();

alter table if exists blog_tags add column if not exists tenant_id uuid;
alter table if exists blog_tags add column if not exists site_id uuid;
alter table if exists blog_tags add column if not exists visibility text not null default 'public';
alter table if exists blog_tags add column if not exists updated_at timestamptz not null default now();

alter table if exists blog_authors add column if not exists tenant_id uuid;
alter table if exists blog_authors add column if not exists site_id uuid;
alter table if exists blog_authors add column if not exists bio text not null default '';
alter table if exists blog_authors add column if not exists updated_at timestamptz not null default now();

alter table if exists newsletter_settings add column if not exists tenant_id uuid;
alter table if exists newsletter_settings add column if not exists site_id uuid;
alter table if exists newsletter_settings add column if not exists send_posts_as_email boolean not null default true;
alter table if exists newsletter_settings add column if not exists settings_json jsonb not null default '{}'::jsonb;
alter table if exists newsletter_settings add column if not exists updated_at timestamptz not null default now();

alter table if exists blog_settings add column if not exists tenant_id uuid;
alter table if exists blog_settings add column if not exists site_id uuid;
alter table if exists blog_settings add column if not exists blog_title text not null default '';
alter table if exists blog_settings add column if not exists description text not null default '';
alter table if exists blog_settings add column if not exists default_author_id uuid;
alter table if exists blog_settings add column if not exists rss_enabled boolean not null default true;
alter table if exists blog_settings add column if not exists comments_enabled boolean not null default true;
alter table if exists blog_settings add column if not exists canonical_url text not null default '';
alter table if exists blog_settings add column if not exists posts_per_page integer not null default 10;
alter table if exists blog_settings add column if not exists social_sharing_enabled boolean not null default true;
alter table if exists blog_settings add column if not exists settings_json jsonb not null default '{}'::jsonb;
alter table if exists blog_settings add column if not exists updated_at timestamptz not null default now();

alter table if exists blog_comments add column if not exists tenant_id uuid;
alter table if exists blog_comments add column if not exists site_id uuid;
alter table if exists blog_comments add column if not exists post_id uuid;
alter table if exists blog_comments add column if not exists author_name text not null default '';
alter table if exists blog_comments add column if not exists author_email text not null default '';
alter table if exists blog_comments add column if not exists content text not null default '';
alter table if exists blog_comments add column if not exists status text not null default 'pending';
alter table if exists blog_comments add column if not exists updated_at timestamptz not null default now();

alter table if exists blog_members add column if not exists tenant_id uuid;
alter table if exists blog_members add column if not exists site_id uuid;
alter table if exists blog_members add column if not exists member_type text not null default 'free';
alter table if exists blog_members add column if not exists joined_at timestamptz not null default now();
alter table if exists blog_members add column if not exists updated_at timestamptz not null default now();

alter table if exists blog_analytics add column if not exists tenant_id uuid;
alter table if exists blog_analytics add column if not exists site_id uuid;
alter table if exists blog_analytics add column if not exists signups integer not null default 0;
alter table if exists blog_analytics add column if not exists newsletter_opens integer not null default 0;
alter table if exists blog_analytics add column if not exists updated_at timestamptz not null default now();

alter table if exists subscription_plans add column if not exists tenant_id uuid;
alter table if exists subscription_plans add column if not exists site_id uuid;
alter table if exists subscription_plans add column if not exists updated_at timestamptz not null default now();

alter table if exists subscription_subscribers add column if not exists tenant_id uuid;
alter table if exists subscription_subscribers add column if not exists site_id uuid;
alter table if exists subscription_subscribers add column if not exists updated_at timestamptz not null default now();

alter table if exists subscriptions add column if not exists tenant_id uuid;
alter table if exists subscriptions add column if not exists site_id uuid;
alter table if exists subscriptions add column if not exists amount numeric(12,2) not null default 0;
alter table if exists subscriptions add column if not exists updated_at timestamptz not null default now();

alter table if exists subscription_invoices add column if not exists tenant_id uuid;
alter table if exists subscription_invoices add column if not exists site_id uuid;
alter table if exists subscription_invoices add column if not exists updated_at timestamptz not null default now();

alter table if exists failed_payments add column if not exists tenant_id uuid;
alter table if exists failed_payments add column if not exists site_id uuid;
alter table if exists failed_payments add column if not exists retry_count integer not null default 0;
alter table if exists failed_payments add column if not exists next_retry_at timestamptz;
alter table if exists failed_payments add column if not exists updated_at timestamptz not null default now();

alter table if exists subscription_settings add column if not exists tenant_id uuid;
alter table if exists subscription_settings add column if not exists site_id uuid;
alter table if exists subscription_settings add column if not exists trial_settings_json jsonb not null default '{}'::jsonb;
alter table if exists subscription_settings add column if not exists settings_json jsonb not null default '{}'::jsonb;
alter table if exists subscription_settings add column if not exists updated_at timestamptz not null default now();

create index if not exists idx_workspace_files_site_id on workspace_files(site_id);
create index if not exists idx_site_releases_site_id on site_releases(site_id);
create index if not exists idx_assets_site_id on assets(site_id);
create index if not exists idx_audit_logs_site_id on audit_logs(site_id);
create index if not exists idx_products_site_id on products(site_id);
create index if not exists idx_orders_site_id on orders(site_id);
create index if not exists idx_customers_site_id on customers(site_id);
create index if not exists idx_coupons_site_id on coupons(site_id);
create index if not exists idx_shipping_zones_site_id on shipping_zones(site_id);
create index if not exists idx_blog_posts_site_id on blog_posts(site_id);
create index if not exists idx_blog_categories_site_id on blog_categories(site_id);
create index if not exists idx_blog_tags_site_id on blog_tags(site_id);
create index if not exists idx_blog_authors_site_id on blog_authors(site_id);
create index if not exists idx_blog_settings_site_id on blog_settings(site_id);
create index if not exists idx_blog_comments_site_id on blog_comments(site_id);
create index if not exists idx_blog_members_site_id on blog_members(site_id);
create index if not exists idx_blog_analytics_site_id on blog_analytics(site_id);
create index if not exists idx_subscription_plans_site_id on subscription_plans(site_id);
create index if not exists idx_subscription_subscribers_site_id on subscription_subscribers(site_id);
create index if not exists idx_subscriptions_site_id on subscriptions(site_id);
create index if not exists idx_subscription_invoices_site_id on subscription_invoices(site_id);
create index if not exists idx_failed_payments_site_id on failed_payments(site_id);
`},
	}
	for _, step := range steps {
		var applied bool
		if err := db.QueryRow(`select exists(select 1 from information_schema.tables where table_name = 'schema_migrations')`).Scan(&applied); err != nil {
			return err
		}
		if applied {
			var exists bool
			if err := db.QueryRow(`select exists(select 1 from schema_migrations where version=$1)`, step.version).Scan(&exists); err == nil && exists {
				continue
			}
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(step.sql); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`insert into schema_migrations(version, applied_at) values($1, now()) on conflict (version) do nothing`, step.version); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func detectSchemaDrift(db *sql.DB) error {
	checks := [][2]string{
		{"tenants", "status"},
		{"sites", "tenant_id"},
		{"workspace_files", "site_id"},
		{"products", "site_id"},
		{"blog_posts", "site_id"},
		{"subscription_plans", "site_id"},
	}
	reset := false
	for _, check := range checks {
		var exists bool
		if err := db.QueryRow(`select exists(select 1 from information_schema.columns where table_name = $1 and column_name = $2)`, check[0], check[1]).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			reset = true
			break
		}
	}
	if !reset {
		return nil
	}
	return fmt.Errorf("schema drift detected; reset the development database and rerun migrations")
}

func seed(db *sql.DB, storageRoot, deliveryBase string) error {
	ctx := context.Background()
	tenantID := ensureUUID("tenant-demo")
	userID := ensureUUID("user-demo")
	siteID := ensureUUID("site-demo")
	passwordHash, err := hashPassword(demoPassword)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `insert into tenants(id,name,slug,status,created_at,updated_at)
		values($1,$2,$3,'active',now(),now())
		on conflict (slug) do update set name=excluded.name,updated_at=now()`,
		tenantID, "Acme Studio", demoTenant); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into users(id,email,name,password_hash,status,created_at,updated_at)
		values($1,$2,$3,$4,'active',now(),now())
		on conflict (email) do update set name=excluded.name,password_hash=excluded.password_hash,updated_at=now()`,
		userID, demoEmail, "Ava Chen", passwordHash); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into tenant_users(id,tenant_id,user_id,role,status,created_at,updated_at)
		values($1,$2,$3,'owner','active',now(),now())
		on conflict (tenant_id, user_id) do update set role='owner',status='active',updated_at=now()`,
		ensureUUID("tenant-user-demo"), tenantID, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into sites(id,tenant_id,name,slug,status,created_at,updated_at)
		values($1,$2,$3,$4,'draft',now(),now())
		on conflict (tenant_id, slug) do update set name=excluded.name,updated_at=now()`,
		siteID, tenantID, "Client landing page", "client-landing-page"); err != nil {
		return err
	}
	if err := seedModules(ctx, tx, tenantID, siteID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	if err := ensureSiteWorkspace(storageRoot, tenantID, siteID); err != nil {
		return err
	}
	if err := ensureSitePublished(storageRoot, tenantID, siteID); err != nil {
		return err
	}
	baseFiles := map[string]string{
		"index.html": defaultWorkspaceHTML("Client landing page"),
		"styles.css": "body { margin: 0; font-family: Arial, sans-serif; background: #f4f7fb; }\n.page { padding: 48px; }\n",
		"app.js":     "console.log('WebsiteBuilder ready');\n",
	}
	for path, content := range baseFiles {
		if err := os.WriteFile(siteWorkspaceFile(storageRoot, tenantID, siteID, path), []byte(content), 0o644); err != nil {
			return err
		}
		if err := syncWorkspaceMetadata(db, tenantID, siteID, path, strings.TrimPrefix(filepath.Ext(path), "."), int64(len(content)), checksum([]byte(content))); err != nil {
			return err
		}
	}
	_, _ = db.Exec(`update sites set published_at = coalesce(published_at, now()) where id=$1 and exists(select 1 from site_releases where site_id=$1)`, siteID)
	_ = writeSeedSnapshot(filepath.Join(storageRoot, "seed_state.json"), seedState{TenantID: tenantID, UserID: userID, SiteID: siteID})
	_ = ensurePublishedCopy(db, storageRoot, deliveryBase, tenantID, siteID, "client-landing-page")
	return nil
}

func seedModules(ctx context.Context, tx *sql.Tx, tenantID, siteID string) error {
	statements := []struct {
		query string
		args  []any
	}{
		{`insert into products(id,tenant_id,site_id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,created_at,updated_at)
			values($1,$2,$3,'Starter kit','starter-kit','Starter product for the site storefront.',49,'SC-001','','General','in_stock','active','Starter kit','Starter product for the site storefront.',now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("product-starter"), tenantID, siteID}},
		{`insert into blog_posts(id,tenant_id,site_id,title,slug,excerpt,content,featured_image_url,author,tags_json,seo_title,seo_description,visibility,status,published_at,created_at,updated_at)
			values($1,$2,$3,'Launching the new site','launching-the-new-site','A short update on the new site rollout.','<p>We are rolling out the new site experience.</p>','','Ava Chen','["launch","product"]'::jsonb,'Launching the new site','A short update on the new site rollout.','public','published',now(),now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("post-launch"), tenantID, siteID}},
		{`insert into subscription_plans(id,tenant_id,site_id,name,description,price,billing_interval,trial_days,status,created_at,updated_at)
			values($1,$2,$3,'Essentials box','Monthly curation for repeat customers.',29,'monthly',7,'active',now(),now())
			on conflict do nothing`, []any{ensureUUID("plan-essentials"), tenantID, siteID}},
		{`insert into orders(id,tenant_id,site_id,order_number,customer_name,customer_email,amount,payment_status,fulfillment_status,status,created_at,updated_at)
			values($1,$2,$3,'1042','Mia Ward','mia@north.co',184,'paid','packed','processing',now(),now())
			on conflict do nothing`, []any{ensureUUID("order-1042"), tenantID, siteID}},
		{`insert into customers(id,tenant_id,site_id,name,email,orders_count,total_spent,last_order_at,created_at,updated_at)
			values($1,$2,$3,'Mia Ward','mia@north.co',3,412,now(),now(),now())
			on conflict do nothing`, []any{ensureUUID("customer-mia"), tenantID, siteID}},
		{`insert into coupons(id,tenant_id,site_id,code,discount_type,value,expires_at,usage_limit,usage_count,status,created_at,updated_at)
			values($1,$2,$3,'WELCOME10','percentage',10,now() + interval '30 days',100,28,'active',now(),now())
			on conflict (site_id, code) do nothing`, []any{ensureUUID("coupon-welcome10"), tenantID, siteID}},
		{`insert into shipping_zones(id,tenant_id,site_id,name,region,method,rate,status,created_at,updated_at)
			values($1,$2,$3,'Domestic','United States','flat_rate',9.99,'active',now(),now())
			on conflict do nothing`, []any{ensureUUID("shipping-domestic"), tenantID, siteID}},
		{`insert into store_settings(id,tenant_id,site_id,currency,store_email,checkout_mode,payment_provider,order_notifications_enabled,settings_json,created_at,updated_at)
			values($1,$2,$3,'USD','orders@sitecraft.app','standard','coming_soon',true,'{}'::jsonb,now(),now())
			on conflict (site_id) do nothing`, []any{ensureUUID("store-settings"), tenantID, siteID}},
		{`insert into tax_settings(id,tenant_id,site_id,taxes_enabled,prices_include_tax,default_region,settings_json,created_at,updated_at)
			values($1,$2,$3,true,false,'US','{}'::jsonb,now(),now())
			on conflict (site_id) do nothing`, []any{ensureUUID("tax-settings"), tenantID, siteID}},
		{`insert into blog_categories(id,tenant_id,site_id,name,slug,description,created_at,updated_at)
			values($1,$2,$3,'Announcements','announcements','Platform and launch updates.',now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("blog-category-announcements"), tenantID, siteID}},
		{`insert into blog_tags(id,tenant_id,site_id,name,slug,visibility,created_at,updated_at)
			values($1,$2,$3,'Launch','launch','public',now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("blog-tag-launch"), tenantID, siteID}},
		{`insert into blog_authors(id,tenant_id,site_id,name,email,role,bio,created_at,updated_at)
			values($1,$2,$3,'Ava Chen','ava@acme-studio.com','Editor','Leads product and content.',now(),now())
			on conflict do nothing`, []any{ensureUUID("blog-author-ava"), tenantID, siteID}},
		{`insert into newsletter_settings(id,tenant_id,site_id,enabled,sender_name,sender_email,send_posts_as_email,settings_json,created_at,updated_at)
			values($1,$2,$3,true,'SiteCraft','hello@sitecraft.app',true,'{}'::jsonb,now(),now())
			on conflict (site_id) do nothing`, []any{ensureUUID("newsletter-settings"), tenantID, siteID}},
		{`insert into blog_settings(id,tenant_id,site_id,blog_title,description,default_author_id,rss_enabled,comments_enabled,canonical_url,posts_per_page,social_sharing_enabled,settings_json,created_at,updated_at)
			values($1,$2,$3,'SiteCraft Journal','News, publishing, and product updates.',$4,true,true,'',10,true,'{}'::jsonb,now(),now())
			on conflict (site_id) do nothing`, []any{ensureUUID("blog-settings"), tenantID, siteID, ensureUUID("blog-author-ava")}},
		{`insert into blog_comments(id,tenant_id,site_id,post_id,author_name,author_email,content,status,created_at,updated_at)
			values($1,$2,$3,$4,'Mia Ward','mia@north.co','Excited to see this launch go live.','pending',now(),now())
			on conflict do nothing`, []any{ensureUUID("blog-comment-launch"), tenantID, siteID, ensureUUID("post-launch")}},
		{`insert into blog_members(id,tenant_id,site_id,name,email,member_type,status,joined_at,created_at,updated_at)
			values($1,$2,$3,'Jordan Lee','jordan@brand.co','paid','active',now(),now(),now())
			on conflict do nothing`, []any{ensureUUID("blog-member-jordan"), tenantID, siteID}},
		{`insert into blog_analytics(id,tenant_id,site_id,post_id,views,signups,newsletter_opens,created_at,updated_at)
			values($1,$2,$3,$4,14800,18,420,now(),now())
			on conflict do nothing`, []any{ensureUUID("blog-analytics"), tenantID, siteID, ensureUUID("post-launch")}},
		{`insert into subscription_subscribers(id,tenant_id,site_id,name,email,status,joined_at,created_at,updated_at)
			values($1,$2,$3,'Riya Thomas','riya@north.co','active',now(),now(),now())
			on conflict do nothing`, []any{ensureUUID("subscriber-riya"), tenantID, siteID}},
		{`insert into subscriptions(id,tenant_id,site_id,subscriber_id,plan_id,status,renewal_date,amount,created_at,updated_at)
			values($1,$2,$3,$4,$5,'active',now() + interval '7 days',29,now(),now())
			on conflict do nothing`, []any{ensureUUID("subscription-riya"), tenantID, siteID, ensureUUID("subscriber-riya"), ensureUUID("plan-essentials")}},
		{`insert into subscription_invoices(id,tenant_id,site_id,invoice_number,subscriber_id,amount,status,issued_at,created_at,updated_at)
			values($1,$2,$3,'INV-203',$4,29,'paid',now(),now(),now())
			on conflict (site_id, invoice_number) do nothing`, []any{ensureUUID("invoice-203"), tenantID, siteID, ensureUUID("subscriber-riya")}},
		{`insert into failed_payments(id,tenant_id,site_id,subscription_id,subscriber_id,amount,status,retry_count,next_retry_at,created_at,updated_at)
			values($1,$2,$3,$4,$5,29,'pending',1,now() + interval '4 days',now(),now())
			on conflict do nothing`, []any{ensureUUID("failed-payment-riya"), tenantID, siteID, ensureUUID("subscription-riya"), ensureUUID("subscriber-riya")}},
		{`insert into subscription_settings(id,tenant_id,site_id,billing_currency,renewal_email_enabled,cancellation_policy,trial_settings_json,settings_json,created_at,updated_at)
			values($1,$2,$3,'USD',true,'Cancel before renewal','{}'::jsonb,'{}'::jsonb,now(),now())
			on conflict (site_id) do nothing`, []any{ensureUUID("subscription-settings"), tenantID, siteID}},
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
			return err
		}
	}
	return nil
}

func writeSeedSnapshot(path string, state seedState) error {
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func ensurePublishedCopy(db *sql.DB, storageRoot, deliveryBase, tenantID, siteID, slug string) error {
	ctx := context.Background()
	if err := ensureSitePublished(storageRoot, tenantID, siteID); err != nil {
		return err
	}
	srcRoot := siteWorkspaceDir(storageRoot, tenantID, siteID)
	dstRoot := sitePublishedDir(storageRoot, tenantID, siteID)
	if err := filepath.Walk(srcRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dstRoot, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}); err != nil {
		return err
	}
	var releaseNumber int
	_ = db.QueryRowContext(ctx, `select coalesce(max(release_number), 0) from site_releases where site_id=$1`, siteID).Scan(&releaseNumber)
	releaseID := uuid.NewString()
	publicURL := strings.TrimRight(deliveryBase, "/") + "/sites/" + slug
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `insert into site_releases(id,tenant_id,site_id,release_number,artifact_path,public_url,status,created_at)
		values($1,$2,$3,$4,$5,$6,'published',now())`,
		releaseID, tenantID, siteID, releaseNumber+1, filepath.ToSlash(dstRoot), publicURL); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `update sites set active_release_id=$1,published_at=now(),status='published',updated_at=now() where id=$2`, releaseID, siteID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `insert into audit_logs(id,tenant_id,site_id,actor_id,action,target_type,target_id,outcome,metadata_json,created_at)
		values($1,$2,$3,null,'publish','site',$4,'success',$5::jsonb,now())`,
		uuid.NewString(), tenantID, siteID, siteID, fmt.Sprintf(`{"public_url":%q}`, publicURL)); err != nil {
		return err
	}
	return tx.Commit()
}

func ensureUUID(seed string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(seed)).String()
}

func seedSnapshot(db *sql.DB) map[string]any {
	var tenants, users, sites int
	_ = db.QueryRow(`select count(*) from tenants`).Scan(&tenants)
	_ = db.QueryRow(`select count(*) from users`).Scan(&users)
	_ = db.QueryRow(`select count(*) from sites`).Scan(&sites)
	return map[string]any{
		"tenants": tenants,
		"users":   users,
		"sites":   sites,
		"defaultCredentials": map[string]string{
			"tenantSlug": demoTenant,
			"email":      demoEmail,
			"password":   demoPassword,
		},
	}
}

func syncWorkspaceMetadata(db *sql.DB, tenantID, siteID, path, fileType string, size int64, sum string) error {
	_, err := db.Exec(`insert into workspace_files(id,tenant_id,site_id,path,file_type,size_bytes,checksum,created_at,updated_at)
		values($1,$2,$3,$4,$5,$6,$7,now(),now())
		on conflict (site_id, path) do update set file_type=excluded.file_type,size_bytes=excluded.size_bytes,checksum=excluded.checksum,updated_at=now()`,
		uuid.NewString(), tenantID, siteID, path, fileType, size, sum)
	return err
}

func readSeedState(root string) (seedState, error) {
	var state seedState
	data, err := os.ReadFile(filepath.Join(root, "seed_state.json"))
	if err != nil {
		return state, err
	}
	err = json.Unmarshal(data, &state)
	return state, err
}

func waitForDatabase(cfg Config, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		db, err := openPostgres(cfg)
		if err == nil {
			_ = db.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(2 * time.Second)
	}
}

func CreateDatabase(cfg Config) error {
	return createDatabaseIfMissing(cfg)
}

func MigrateDatabase(cfg Config) error {
	db, err := openPostgres(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	return migrate(db)
}

func SeedDatabase(cfg Config) error {
	db, err := openPostgres(cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	return seed(db, cfg.StorageRoot, cfg.deliveryURL())
}

func InspectDatabase(cfg Config) (map[string]any, error) {
	db, err := openPostgres(cfg)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	result := map[string]any{}
	for _, table := range []string{"tenants", "users", "tenant_users", "sites", "workspace_files", "site_releases", "products", "blog_posts", "subscription_plans"} {
		var exists bool
		if err := db.QueryRow(`select exists(select 1 from information_schema.tables where table_name = $1)`, table).Scan(&exists); err != nil {
			return nil, err
		}
		result[table] = exists
	}
	result["snapshot"] = seedSnapshot(db)
	return result, nil
}
