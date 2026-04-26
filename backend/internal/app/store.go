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

type demoSiteDefinition struct {
	IDSeed string
	Name   string
	Slug   string
}

type demoNavLink struct {
	Label string
	Href  string
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
create index if not exists idx_sites_active_release_id on sites(active_release_id);
create index if not exists idx_sites_status_slug_id on sites(status, slug, id);
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
		{"20260425_004_blog_schema_backfill", `
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

create index if not exists idx_blog_settings_site_id on blog_settings(site_id);
create index if not exists idx_blog_comments_site_id on blog_comments(site_id);
`},
		{"20260425_005_delivery_indexes", `
create index if not exists idx_sites_active_release_id on sites(active_release_id);
create index if not exists idx_sites_status_slug_id on sites(status, slug, id);
create index if not exists idx_site_releases_site_status on site_releases(site_id, status, created_at desc);
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
	demoSites := []demoSiteDefinition{
		{IDSeed: "site-client-advisory-studio", Name: "Client Advisory Studio", Slug: "client-advisory-studio"},
		{IDSeed: "site-maison-vela", Name: "Maison Vela", Slug: "maison-vela"},
		{IDSeed: "site-pawnest-club", Name: "PawNest Club", Slug: "pawnest-club"},
		{IDSeed: "site-printforge-studio", Name: "PrintForge Studio", Slug: "printforge-studio"},
		{IDSeed: "site-pixelshelf", Name: "PixelShelf", Slug: "pixelshelf"},
	}
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
	for _, site := range demoSites {
		if _, err := tx.ExecContext(ctx, `insert into sites(id,tenant_id,name,slug,status,created_at,updated_at)
			values($1,$2,$3,$4,'draft',now(),now())
			on conflict (tenant_id, slug) do update set name=excluded.name,updated_at=now()`,
			ensureUUID(site.IDSeed), tenantID, site.Name, site.Slug); err != nil {
			return err
		}
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
	if err := ensureSeedWorkspace(storageRoot, tenantID, siteID); err != nil {
		return err
	}
	if err := syncWorkspaceTreeMetadata(db, storageRoot, tenantID, siteID); err != nil {
		return err
	}
	_, _ = db.Exec(`update sites set published_at = coalesce(published_at, now()) where id=$1 and exists(select 1 from site_releases where site_id=$1)`, siteID)
	_ = writeSeedSnapshot(filepath.Join(storageRoot, "seed_state.json"), seedState{TenantID: tenantID, UserID: userID, SiteID: siteID})
	_ = ensurePublishedCopy(db, storageRoot, deliveryBase, tenantID, siteID, "client-landing-page")
	for _, site := range demoSites {
		resolvedSiteID, err := resolveSiteIDBySlug(ctx, db, tenantID, site.Slug)
		if err != nil {
			return err
		}
		if err := ensureSiteWorkspace(storageRoot, tenantID, resolvedSiteID); err != nil {
			return err
		}
		if err := ensureSitePublished(storageRoot, tenantID, resolvedSiteID); err != nil {
			return err
		}
		if err := writeDemoWorkspace(storageRoot, tenantID, resolvedSiteID, site); err != nil {
			return err
		}
		if err := syncWorkspaceTreeMetadata(db, storageRoot, tenantID, resolvedSiteID); err != nil {
			return err
		}
		if err := ensurePublishedCopy(db, storageRoot, deliveryBase, tenantID, resolvedSiteID, site.Slug); err != nil {
			return err
		}
	}
	return nil
}

func ensureSeedWorkspace(storageRoot, tenantID, siteID string) error {
	workspace := siteWorkspaceDir(storageRoot, tenantID, siteID)
	entries, err := os.ReadDir(workspace)
	if err == nil && len(entries) > 0 {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	files := map[string]string{
		"index.html": defaultWorkspaceHTML("Client landing page"),
		"styles.css": "body { margin: 0; font-family: Arial, sans-serif; background: #f4f7fb; }\n.page { padding: 48px; }\n",
		"script.js":  "console.log('SiteCraft ready');\n",
	}
	for path, content := range files {
		target := siteWorkspaceFile(storageRoot, tenantID, siteID, path)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func syncWorkspaceTreeMetadata(db *sql.DB, storageRoot, tenantID, siteID string) error {
	root := siteWorkspaceDir(storageRoot, tenantID, siteID)
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return syncWorkspaceMetadata(db, tenantID, siteID, rel, strings.TrimPrefix(filepath.Ext(rel), "."), info.Size(), checksum(content))
	})
}

func seedModules(ctx context.Context, tx *sql.Tx, tenantID, siteID string) error {
	statements := []struct {
		query string
		args  []any
	}{
		{`insert into products(id,tenant_id,site_id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,created_at,updated_at)
			values($1,$2,$3,'Starter kit','starter-kit','Starter product for the site storefront.',49,'SC-001','','General','in_stock','active','Starter kit','Starter product for the site storefront.',now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("product-starter"), tenantID, siteID}},
		{`insert into products(id,tenant_id,site_id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,created_at,updated_at)
			values($1,$2,$3,'Growth dashboard package','growth-dashboard-package','Executive KPI board and advisory reporting kit.',1499,'SC-002','','Analytics','in_stock','active','Growth dashboard package','Executive KPI board and advisory reporting kit.',now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("product-growth-dashboard"), tenantID, siteID}},
		{`insert into products(id,tenant_id,site_id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,created_at,updated_at)
			values($1,$2,$3,'Operator playbook','operator-playbook','Workflow blueprint for marketing, sales, and delivery teams.',899,'SC-003','','Operations','in_stock','active','Operator playbook','Workflow blueprint for marketing, sales, and delivery teams.',now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("product-operator-playbook"), tenantID, siteID}},
		{`insert into products(id,tenant_id,site_id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,created_at,updated_at)
			values($1,$2,$3,'Messaging architecture sprint','messaging-architecture-sprint','Homepage and offer messaging system for consulting brands.',1299,'SC-004','','Strategy','in_stock','active','Messaging architecture sprint','Homepage and offer messaging system for consulting brands.',now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("product-messaging-architecture"), tenantID, siteID}},
		{`insert into products(id,tenant_id,site_id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,created_at,updated_at)
			values($1,$2,$3,'Automation workflow blueprint','automation-workflow-blueprint','Service delivery workflow map with automation handoff recommendations.',1799,'SC-005','','Automation','in_stock','active','Automation workflow blueprint','Service delivery workflow map with automation handoff recommendations.',now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("product-automation-workflow"), tenantID, siteID}},
		{`insert into products(id,tenant_id,site_id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,created_at,updated_at)
			values($1,$2,$3,'Website conversion framework','website-conversion-framework','Consulting website page system focused on proof, positioning, and CTA flow.',1599,'SC-006','','Website','in_stock','active','Website conversion framework','Consulting website page system focused on proof, positioning, and CTA flow.',now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("product-website-conversion"), tenantID, siteID}},
		{`insert into products(id,tenant_id,site_id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,created_at,updated_at)
			values($1,$2,$3,'Leadership metrics cockpit','leadership-metrics-cockpit','Executive analytics layer for pipeline, delivery, and retention performance.',1899,'SC-007','','Analytics','in_stock','active','Leadership metrics cockpit','Executive analytics layer for pipeline, delivery, and retention performance.',now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("product-leadership-metrics"), tenantID, siteID}},
		{`insert into products(id,tenant_id,site_id,name,slug,description,price,sku,image_url,category,stock_status,status,seo_title,seo_description,created_at,updated_at)
			values($1,$2,$3,'Offer positioning review','offer-positioning-review','Compact advisory package for clarifying your market-facing offer and value story.',699,'SC-008','','Strategy','in_stock','active','Offer positioning review','Compact advisory package for clarifying your market-facing offer and value story.',now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("product-offer-positioning"), tenantID, siteID}},
		{`insert into blog_posts(id,tenant_id,site_id,title,slug,excerpt,content,featured_image_url,author,tags_json,seo_title,seo_description,visibility,status,published_at,created_at,updated_at)
			values($1,$2,$3,'Launching the new site','launching-the-new-site','A short update on the new site rollout.','<p>We are rolling out the new site experience.</p>','','Ava Chen','["launch","product"]'::jsonb,'Launching the new site','A short update on the new site rollout.','public','published',now(),now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("post-launch"), tenantID, siteID}},
		{`insert into blog_posts(id,tenant_id,site_id,title,slug,excerpt,content,featured_image_url,author,tags_json,seo_title,seo_description,visibility,status,published_at,created_at,updated_at)
			values($1,$2,$3,'How consulting teams scale delivery','how-consulting-teams-scale-delivery','A practical note on scaling delivery quality across growing client work.','<p>Document your handoffs, simplify approvals, and make delivery visible.</p>','','Maya Rowe','["operations","delivery"]'::jsonb,'How consulting teams scale delivery','A practical note on scaling delivery quality across growing client work.','public','published',now(),now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("post-scale-delivery"), tenantID, siteID}},
		{`insert into blog_posts(id,tenant_id,site_id,title,slug,excerpt,content,featured_image_url,author,tags_json,seo_title,seo_description,visibility,status,published_at,created_at,updated_at)
			values($1,$2,$3,'Dashboards leaders actually use','dashboards-leaders-actually-use','Why operating dashboards should reduce friction instead of creating noise.','<p>Focus on decisions, exceptions, and next actions instead of vanity metrics.</p>','','Jordan Lee','["analytics","leadership"]'::jsonb,'Dashboards leaders actually use','Why operating dashboards should reduce friction instead of creating noise.','public','published',now(),now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("post-dashboards"), tenantID, siteID}},
		{`insert into blog_posts(id,tenant_id,site_id,title,slug,excerpt,content,featured_image_url,author,tags_json,seo_title,seo_description,visibility,status,published_at,created_at,updated_at)
			values($1,$2,$3,'Positioning a consulting firm for the next stage','positioning-a-consulting-firm-for-the-next-stage','How mature firms refresh messaging without losing credibility with existing clients.','<p>Positioning work should make your pipeline clearer, not louder.</p>','','Ava Chen','["positioning","strategy"]'::jsonb,'Positioning a consulting firm for the next stage','How mature firms refresh messaging without losing credibility with existing clients.','public','published',now(),now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("post-positioning-next-stage"), tenantID, siteID}},
		{`insert into blog_posts(id,tenant_id,site_id,title,slug,excerpt,content,featured_image_url,author,tags_json,seo_title,seo_description,visibility,status,published_at,created_at,updated_at)
			values($1,$2,$3,'What automation should fix first','what-automation-should-fix-first','A practical order of operations for automating intake, delivery, and reporting.','<p>Start with repeatable decisions and measurable handoffs before chasing tool sprawl.</p>','','Maya Rowe','["automation","operations"]'::jsonb,'What automation should fix first','A practical order of operations for automating intake, delivery, and reporting.','public','published',now(),now(),now())
			on conflict (site_id, slug) do nothing`, []any{ensureUUID("post-automation-first"), tenantID, siteID}},
		{`insert into subscription_plans(id,tenant_id,site_id,name,description,price,billing_interval,trial_days,status,created_at,updated_at)
			values($1,$2,$3,'Essentials box','Monthly curation for repeat customers.',29,'monthly',7,'active',now(),now())
			on conflict do nothing`, []any{ensureUUID("plan-essentials"), tenantID, siteID}},
		{`insert into subscription_plans(id,tenant_id,site_id,name,description,price,billing_interval,trial_days,status,created_at,updated_at)
			values($1,$2,$3,'Growth review','Monthly operating review with KPI commentary.',149,'monthly',14,'active',now(),now())
			on conflict do nothing`, []any{ensureUUID("plan-growth-review"), tenantID, siteID}},
		{`insert into subscription_plans(id,tenant_id,site_id,name,description,price,billing_interval,trial_days,status,created_at,updated_at)
			values($1,$2,$3,'Embedded advisor','Quarterly strategic support for leadership teams.',499,'monthly',14,'active',now(),now())
			on conflict do nothing`, []any{ensureUUID("plan-embedded-advisor"), tenantID, siteID}},
		{`insert into subscription_plans(id,tenant_id,site_id,name,description,price,billing_interval,trial_days,status,created_at,updated_at)
			values($1,$2,$3,'Annual operating partner','Year-round advisory cadence with board-level planning support.',4999,'yearly',21,'active',now(),now())
			on conflict do nothing`, []any{ensureUUID("plan-annual-operating-partner"), tenantID, siteID}},
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
	if err := ensureSiteReleases(storageRoot, tenantID, siteID); err != nil {
		return err
	}
	srcRoot := siteWorkspaceDir(storageRoot, tenantID, siteID)
	var releaseNumber int
	_ = db.QueryRowContext(ctx, `select coalesce(max(release_number), 0) from site_releases where site_id=$1`, siteID).Scan(&releaseNumber)
	releaseID := uuid.NewString()
	dstRoot := siteReleaseDir(storageRoot, tenantID, siteID, releaseID)
	renderCtx, err := buildRenderContextFromDB(ctx, db, deliveryBase, tenantID, siteID, slug)
	if err != nil {
		return err
	}
	renderErrors, err := renderWorkspaceDir(srcRoot, dstRoot, renderCtx)
	if err != nil {
		return err
	}
	if len(renderErrors) > 0 {
		_ = os.RemoveAll(dstRoot)
		return fmt.Errorf("seed release render failed")
	}
	if err := copyDir(siteWorkspaceDir(storageRoot, tenantID, siteID), sitePublishedDir(storageRoot, tenantID, siteID)); err != nil {
		return err
	}
	publicURL := strings.TrimRight(deliveryBase, "/") + "/sites/" + siteID + "/" + slug
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `insert into site_releases(id,tenant_id,site_id,release_number,artifact_path,public_url,status,created_at)
		values($1,$2,$3,$4,$5,$6,'published',now())`,
		releaseID, tenantID, siteID, releaseNumber+1, filepath.ToSlash(filepath.Join("sites", tenantID, siteID, "releases", releaseID)), publicURL); err != nil {
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

func resolveSiteIDBySlug(ctx context.Context, db *sql.DB, tenantID, slug string) (string, error) {
	var siteID string
	err := db.QueryRowContext(ctx, `select id from sites where tenant_id=$1 and slug=$2`, tenantID, slug).Scan(&siteID)
	return siteID, err
}

func writeDemoWorkspace(storageRoot, tenantID, siteID string, site demoSiteDefinition) error {
	files := demoWorkspaceFiles(site)
	for path, content := range files {
		target := siteWorkspaceFile(storageRoot, tenantID, siteID, path)
		if err := atomicWriteFile(target, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func demoWorkspaceFiles(site demoSiteDefinition) map[string]string {
	switch site.Slug {
	case "client-advisory-studio":
		return consultingDemoWorkspace(site.Name)
	case "maison-vela":
		return fashionDemoWorkspace(site.Name)
	case "pawnest-club":
		return petSubscriptionWorkspace(site.Name)
	case "printforge-studio":
		return customProductsWorkspace(site.Name)
	case "pixelshelf":
		return digitalProductsWorkspace(site.Name)
	default:
		return map[string]string{"index.html": defaultWorkspaceHTML(site.Name)}
	}
}

func pageHTML(title, description, brand string, links []demoNavLink, body string, cart bool) string {
	var nav strings.Builder
	for _, link := range links {
		nav.WriteString(`<a href="` + link.Href + `">` + link.Label + `</a>`)
	}
	cartMarkup := ""
	if cart {
		cartMarkup = `<button class="cart-button" type="button" data-cart-toggle>Cart (<span id="cart-count">0</span>)</button>`
	}
	return "<!doctype html>\n<html lang=\"en\">\n  <head>\n    <meta charset=\"UTF-8\" />\n    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\" />\n    <title>" + title + "</title>\n    <meta name=\"description\" content=\"" + description + "\" />\n    <link rel=\"stylesheet\" href=\"styles.css\" />\n    <script src=\"script.js\" defer></script>\n  </head>\n  <body>\n    <div class=\"site-shell\">\n      <header class=\"site-header\">\n        <div class=\"container nav-row\">\n          <a class=\"brand\" href=\"index.html\"><span class=\"brand-mark\"></span><span>" + brand + "</span></a>\n          <nav class=\"site-nav\">" + nav.String() + cartMarkup + "</nav>\n        </div>\n      </header>\n      <main>" + body + "</main>\n      <footer class=\"site-footer\">\n        <div class=\"container footer-grid\">\n          <div><strong>" + brand + "</strong><p>Professional demo website built to test editing, preview, publish, and delivery flows in SiteCraft.</p></div>\n          <div><p>Developer-first HTML workspace. Responsive layout. Frontend-only interactions where needed.</p></div>\n        </div>\n      </footer>\n    </div>\n    <aside class=\"cart-drawer\" data-cart-drawer hidden>\n      <div class=\"cart-head\"><h2>Cart</h2><button type=\"button\" class=\"mini-ghost\" data-cart-close>Close</button></div>\n      <div class=\"cart-items\" data-cart-items></div>\n      <div class=\"cart-empty\" data-cart-empty>Your cart is empty.</div>\n      <div class=\"cart-footer\"><div class=\"subtotal-row\"><span>Subtotal</span><strong data-cart-subtotal>INR 0.00</strong></div><button type=\"button\" class=\"primary-button\" data-fake-checkout>Checkout</button></div>\n    </aside>\n  </body>\n</html>\n"
}

func defaultStoreScript(siteKey string) string {
	return `(function () {
  var body = document.body;
  var current = (location.pathname.split('/').pop() || 'index.html').toLowerCase();
  document.querySelectorAll('.site-nav a').forEach(function (link) {
    if ((link.getAttribute('href') || '').toLowerCase() === current) link.classList.add('active');
  });

  var cartKey = 'sitecraft-demo-cart-` + siteKey + `';
  function readCart() {
    try { return JSON.parse(localStorage.getItem(cartKey) || '[]'); } catch (e) { return []; }
  }
  function writeCart(items) { localStorage.setItem(cartKey, JSON.stringify(items)); }
  function money(value) { return 'INR ' + Number(value || 0).toFixed(2); }
  function renderCart() {
    var items = readCart();
    var count = items.reduce(function (sum, item) { return sum + item.qty; }, 0);
    var subtotal = items.reduce(function (sum, item) { return sum + (item.price * item.qty); }, 0);
    var countNode = document.getElementById('cart-count');
    if (countNode) countNode.textContent = String(count);
    var itemsHost = document.querySelector('[data-cart-items]');
    var emptyState = document.querySelector('[data-cart-empty]');
    var subtotalNode = document.querySelector('[data-cart-subtotal]');
    if (subtotalNode) subtotalNode.textContent = money(subtotal);
    if (!itemsHost || !emptyState) return;
    itemsHost.innerHTML = '';
    if (!items.length) {
      emptyState.hidden = false;
      return;
    }
    emptyState.hidden = true;
    items.forEach(function (item, index) {
      var row = document.createElement('div');
      row.className = 'cart-line';
      row.innerHTML = '<div><strong>' + item.name + '</strong><span>' + money(item.price) + '</span></div>' +
        '<div class="qty-controls"><button type="button" data-cart-minus="' + index + '">-</button><span>' + item.qty + '</span><button type="button" data-cart-plus="' + index + '">+</button><button type="button" data-cart-remove="' + index + '">Remove</button></div>';
      itemsHost.appendChild(row);
    });
  }
  function mutateCart(mutator) {
    var next = readCart();
    mutator(next);
    writeCart(next.filter(function (item) { return item.qty > 0; }));
    renderCart();
  }
  document.addEventListener('click', function (event) {
    var add = event.target.closest('[data-add-to-cart]');
    if (add) {
      mutateCart(function (items) {
        var name = add.getAttribute('data-name') || 'Item';
        var price = Number(add.getAttribute('data-price') || 0);
        var found = items.find(function (item) { return item.name === name; });
        if (found) found.qty += 1; else items.push({ name: name, price: price, qty: 1 });
      });
      var drawer = document.querySelector('[data-cart-drawer]');
      if (drawer) drawer.hidden = false;
      return;
    }
    if (event.target.closest('[data-cart-toggle]')) {
      var drawer = document.querySelector('[data-cart-drawer]');
      if (drawer) drawer.hidden = !drawer.hidden;
      return;
    }
    if (event.target.closest('[data-cart-close]')) {
      var drawer = document.querySelector('[data-cart-drawer]');
      if (drawer) drawer.hidden = true;
      return;
    }
    if (event.target.matches('[data-cart-minus]')) {
      var minusIndex = Number(event.target.getAttribute('data-cart-minus'));
      mutateCart(function (items) { if (items[minusIndex]) items[minusIndex].qty -= 1; });
      return;
    }
    if (event.target.matches('[data-cart-plus]')) {
      var plusIndex = Number(event.target.getAttribute('data-cart-plus'));
      mutateCart(function (items) { if (items[plusIndex]) items[plusIndex].qty += 1; });
      return;
    }
    if (event.target.matches('[data-cart-remove]')) {
      var removeIndex = Number(event.target.getAttribute('data-cart-remove'));
      mutateCart(function (items) { if (items[removeIndex]) items.splice(removeIndex, 1); });
      return;
    }
    if (event.target.closest('[data-fake-checkout]')) {
      alert('Checkout is a frontend-only demo in this test site.');
    }
  });
  document.querySelectorAll('[data-filter-input]').forEach(function (input) {
    input.addEventListener('input', applyFilters);
  });
  document.querySelectorAll('[data-filter-select]').forEach(function (input) {
    input.addEventListener('change', applyFilters);
  });
  function applyFilters() {
    var term = (document.querySelector('[data-filter-input]') || {}).value || '';
    var category = (document.querySelector('[data-filter-category]') || {}).value || '';
    var extra = (document.querySelector('[data-filter-extra]') || {}).value || '';
    var sort = (document.querySelector('[data-filter-sort]') || {}).value || '';
    var cards = Array.prototype.slice.call(document.querySelectorAll('[data-product-card]'));
    cards.forEach(function (card) {
      var ok = true;
      var name = (card.getAttribute('data-name') || '').toLowerCase();
      var cardCategory = card.getAttribute('data-category') || '';
      var tag = card.getAttribute('data-extra') || '';
      if (term && name.indexOf(term.toLowerCase()) === -1) ok = false;
      if (category && cardCategory !== category) ok = false;
      if (extra && tag !== extra) ok = false;
      card.hidden = !ok;
    });
    var grid = document.querySelector('[data-product-grid]');
    if (grid && sort) {
      cards.sort(function (a, b) {
        var aPrice = Number(a.getAttribute('data-price') || 0);
        var bPrice = Number(b.getAttribute('data-price') || 0);
        return sort === 'high' ? bPrice - aPrice : aPrice - bPrice;
      }).forEach(function (card) { grid.appendChild(card); });
    }
  }
  document.querySelectorAll('[data-demo-form]').forEach(function (form) {
    form.addEventListener('submit', function (event) {
      event.preventDefault();
      var target = form.querySelector('[data-form-success]');
      if (target) target.hidden = false;
      form.reset();
    });
  });
  renderCart();
  applyFilters();
})();` + "\n"
}

func siteThemeCSS(accent, surface, ink string) string {
	return `:root{--bg:#f4f5f7;--surface:` + surface + `;--surface-2:#eef1f4;--text:` + ink + `;--muted:#5f6773;--accent:` + accent + `;--border:#d5dce4;--radius:6px;--shadow:0 18px 46px rgba(17,24,39,.08)}*{box-sizing:border-box}html,body{margin:0;padding:0}body{font-family:Arial,sans-serif;background:linear-gradient(180deg,#fafbfd 0%,#f1f4f8 100%);color:var(--text);line-height:1.5}a{text-decoration:none;color:inherit}.container{width:min(1120px,calc(100% - 32px));margin:0 auto}.site-header{position:sticky;top:0;z-index:10;background:rgba(255,255,255,.96);border-bottom:1px solid var(--border);backdrop-filter:blur(12px)}.nav-row{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:16px 0}.brand{display:flex;align-items:center;gap:10px;font-weight:700}.brand-mark{width:14px;height:14px;background:var(--accent);display:inline-block;border-radius:3px}.site-nav{display:flex;flex-wrap:wrap;gap:14px;align-items:center}.site-nav a{padding:8px 0;color:var(--muted)}.site-nav a.active,.site-nav a:hover{color:var(--text)}.cart-button,.primary-button,.secondary-button,.mini-ghost{border:1px solid var(--border);background:#fff;color:var(--text);padding:10px 14px;border-radius:var(--radius);font:inherit;cursor:pointer}.primary-button{background:var(--accent);border-color:var(--accent);color:#fff}.hero,.page-hero{padding:72px 0 40px}.hero-grid,.two-column,.pricing-grid,.card-grid,.metrics-grid,.store-layout,.footer-grid,.filter-row{display:grid;gap:18px}.hero-grid{grid-template-columns:1.2fr .8fr;align-items:center}.metrics-grid{grid-template-columns:repeat(3,minmax(0,1fr));margin-top:18px}.card-grid{grid-template-columns:repeat(3,minmax(0,1fr))}.two-column,.store-layout,.footer-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.pricing-grid{grid-template-columns:repeat(3,minmax(0,1fr))}.section{padding:28px 0}.eyebrow{color:var(--accent);text-transform:uppercase;letter-spacing:.14em;font-size:.76rem;font-weight:700}.hero h1,.page-hero h1{font-size:clamp(2rem,5vw,4rem);line-height:1.02;margin:10px 0 16px}.lead{font-size:1.02rem;color:var(--muted);max-width:60ch}.panel,.card,.price-card,.plan-card,.product-card,.faq-item,.metric,.contact-card{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);box-shadow:var(--shadow);padding:20px}.metric strong{display:block;font-size:1.6rem}.tag,.badge{display:inline-flex;align-items:center;padding:5px 9px;border-radius:999px;background:#eef3ff;color:var(--accent);font-size:.76rem;font-weight:700}.product-card{display:grid;gap:10px}.product-card strong,.price-card strong,.plan-card strong{font-size:1.2rem}.product-card button,.plan-card button{justify-self:start}.filter-row{grid-template-columns:2fr 1fr 1fr 1fr;margin-bottom:18px}.filter-row input,.filter-row select,.contact-form input,.contact-form textarea,.customizer select,.customizer input{width:100%;padding:11px 12px;border:1px solid var(--border);border-radius:var(--radius);font:inherit;background:#fff}.contact-form,.customizer{display:grid;gap:12px}.contact-form textarea{min-height:150px;resize:vertical}.site-footer{padding:26px 0;border-top:1px solid var(--border);margin-top:40px;background:#fff}.footer-grid p{color:var(--muted);margin:8px 0 0}.cart-drawer{position:fixed;top:0;right:0;width:min(360px,100%);height:100vh;background:#fff;border-left:1px solid var(--border);box-shadow:-16px 0 40px rgba(17,24,39,.12);padding:18px;display:grid;grid-template-rows:auto 1fr auto;gap:16px;z-index:40}.cart-head,.subtotal-row,.qty-controls{display:flex;align-items:center;justify-content:space-between;gap:10px}.cart-items{display:grid;gap:12px;overflow:auto}.cart-line{display:grid;gap:8px;padding-bottom:12px;border-bottom:1px solid var(--border)}.qty-controls button{padding:6px 10px}.cta-band{background:var(--accent);color:#fff;padding:24px;border-radius:var(--radius);display:flex;justify-content:space-between;gap:16px;align-items:center}.table{width:100%;border-collapse:collapse}.table th,.table td{padding:12px;border-bottom:1px solid var(--border);text-align:left}.success-message{padding:12px 14px;background:#e8fff1;border:1px solid #b7ebcc;border-radius:var(--radius)}.hero-visual{min-height:280px;border-radius:var(--radius);background:linear-gradient(135deg,var(--accent),#1f2937);box-shadow:var(--shadow)}.quote{font-style:italic}.feature-list{display:grid;gap:10px;padding:0;margin:0;list-style:none}.feature-list li{padding-left:18px;position:relative}.feature-list li:before{content:"";position:absolute;left:0;top:.55em;width:8px;height:8px;background:var(--accent);border-radius:2px}@media(max-width:980px){.hero-grid,.card-grid,.pricing-grid,.two-column,.store-layout,.footer-grid,.metrics-grid,.filter-row{grid-template-columns:1fr}}`
}

func consultingDemoWorkspace(brand string) map[string]string {
	links := []demoNavLink{{"Home", "index.html"}, {"About", "about.html"}, {"Services", "services.html"}, {"Case Studies", "case-studies.html"}, {"Pricing", "pricing.html"}, {"Contact", "contact.html"}}
	return map[string]string{
		"index.html":        pageHTML(brand+" | Advisory for digital growth", "Consulting demo site for SiteCraft.", brand, links, `<section class="hero"><div class="container hero-grid"><div><p class="eyebrow">Consulting website</p><h1>Advisory, systems, and digital execution for service firms in motion.</h1><p class="lead">`+brand+` helps leadership teams sharpen positioning, modernize operations, and launch polished digital experiences that support growth.</p><div class="metrics-grid"><article class="metric"><strong>38%</strong><span>pipeline lift after repositioning</span></article><article class="metric"><strong>11 weeks</strong><span>average transformation sprint</span></article><article class="metric"><strong>92%</strong><span>retainer renewal across advisory clients</span></article></div></div><div class="hero-visual"></div></div></section><section class="section"><div class="container card-grid"><article class="card"><span class="tag">Strategy</span><h3>Positioning and offer clarity</h3><p>Turn complex delivery into a message buyers understand fast.</p></article><article class="card"><span class="tag">Operations</span><h3>Workflow and handoff design</h3><p>Reduce delivery drag with cleaner internal systems and reporting.</p></article><article class="card"><span class="tag">Digital</span><h3>Launch-ready website systems</h3><p>Build a credible front door that matches the quality of your client work.</p></article></div></section><section class="section"><div class="container cta-band"><div><p class="eyebrow">Featured outcome</p><h2>From scattered service lines to a cohesive go-to-market story.</h2></div><a class="primary-button" href="case-studies.html">View case studies</a></div></section>`, false),
		"about.html":        pageHTML("About | "+brand, "About the consulting demo site.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">About</p><h1>Built for firms that have outgrown yesterday’s message and systems.</h1><p class="lead">We combine business advisory, operating design, and website strategy so teams can move from reactive growth to repeatable growth.</p></div></section><section class="section"><div class="container two-column"><article class="card"><h3>Mission</h3><p>Help ambitious service firms align story, systems, and delivery quality.</p><ul class="feature-list"><li>Clear market positioning</li><li>Sharper internal handoffs</li><li>Web experiences that convert qualified buyers</li></ul></article><article class="card"><h3>Operating principles</h3><ul class="feature-list"><li>Strategy must survive execution.</li><li>Systems should reduce manager load.</li><li>Every page should support trust and action.</li></ul></article></div></section>`, false),
		"services.html":     pageHTML("Services | "+brand, "Service offerings for the consulting demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Services</p><h1>Consulting engagements designed for momentum, clarity, and measurable traction.</h1></div></section><section class="section"><div class="container card-grid"><article class="card"><h3>Business strategy</h3><p>Refine offers, packaging, and sales narrative.</p></article><article class="card"><h3>Automation advisory</h3><p>Map repeatable workflows and remove admin friction.</p></article><article class="card"><h3>Website advisory</h3><p>Plan high-trust pages, conversion structure, and proof architecture.</p></article><article class="card"><h3>Analytics</h3><p>Define the KPIs leadership teams will actually use.</p></article><article class="card"><h3>Operating systems</h3><p>Clarify approvals, ownership, and delivery cadence.</p></article><article class="card"><h3>Implementation support</h3><p>Translate recommendations into launch-ready execution.</p></article></div></section>`, false),
		"case-studies.html": pageHTML("Case Studies | "+brand, "Case study page for the consulting demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Case studies</p><h1>Three concise proof points that test multi-page publishing and polished presentation.</h1></div></section><section class="section"><div class="container card-grid"><article class="card"><span class="tag">B2B services</span><h3>Offer redesign for a delivery-heavy consultancy</h3><p>Improved homepage clarity and lifted qualified leads by 38% in one quarter.</p></article><article class="card"><span class="tag">Professional services</span><h3>Ops reset for an overstretched leadership team</h3><p>Standardized reporting, removed duplicate approvals, and shortened weekly ops meetings by 40%.</p></article><article class="card"><span class="tag">Advisory</span><h3>Website launch for a niche growth partner</h3><p>Reframed the offer, rebuilt the page structure, and supported a cleaner close path.</p></article></div></section>`, false),
		"pricing.html":      pageHTML("Pricing | "+brand, "Pricing page for the consulting demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Pricing</p><h1>Three packages to test pricing layouts, comparison tables, and CTA flow.</h1></div></section><section class="section"><div class="container pricing-grid"><article class="price-card"><span class="tag">Sprint</span><h3>Strategy Sprint</h3><strong>INR 49,000</strong><p>Fast clarity on offer, story, and homepage structure.</p></article><article class="price-card"><span class="tag">Build</span><h3>Growth System</h3><strong>INR 1,25,000</strong><p>Messaging, systems map, and launch-ready conversion architecture.</p></article><article class="price-card"><span class="tag">Partner</span><h3>Operating Partner</h3><strong>INR 2,40,000</strong><p>Embedded advisory for teams executing a broader transformation.</p></article></div></section>`, false),
		"contact.html":      pageHTML("Contact | "+brand, "Contact page for the consulting demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Contact</p><h1>Test form UI, JS success handling, and responsive layout.</h1></div></section><section class="section"><div class="container two-column"><article class="contact-card"><h3>Start a conversation</h3><form class="contact-form" data-demo-form><input type="text" placeholder="Name" required /><input type="email" placeholder="Email" required /><input type="text" placeholder="Company" /><textarea placeholder="What are you trying to improve?"></textarea><button class="primary-button" type="submit">Send request</button><p class="success-message" data-form-success hidden>Thanks. This is a frontend-only success state for platform testing.</p></form></article><article class="card"><h3>Contact details</h3><p>hello@clientadvisorystudio.test</p><p>Bangalore, India</p><p class="quote">Use this page to test save, preview, publish, and live form styling.</p></article></div></section>`, false),
		"styles.css":        siteThemeCSS("#0f5fe8", "#ffffff", "#111827"),
		"script.js":         defaultStoreScript("client-advisory-studio"),
	}
}

func fashionDemoWorkspace(brand string) map[string]string {
	links := []demoNavLink{{"Home", "index.html"}, {"About", "about.html"}, {"Shop", "shop.html"}, {"Collections", "collections.html"}, {"Contact", "contact.html"}}
	shopBody := `<section class="page-hero"><div class="container"><p class="eyebrow">Fashion store</p><h1>Searchable product grid with filters, cart, and compact merchandising.</h1></div></section><section class="section"><div class="container"><div class="filter-row"><input data-filter-input type="search" placeholder="Search products" /><select data-filter-select data-filter-category><option value="">All categories</option><option value="Shirts">Shirts</option><option value="Outerwear">Outerwear</option><option value="Accessories">Accessories</option><option value="Bottoms">Bottoms</option></select><select data-filter-select data-filter-extra><option value="">All sizes</option><option value="S">Size S</option><option value="M">Size M</option><option value="L">Size L</option></select><select data-filter-select data-filter-sort><option value="">Sort</option><option value="low">Price low to high</option><option value="high">Price high to low</option></select></div><div class="card-grid" data-product-grid><article class="product-card" data-product-card data-name="Linen shirt" data-category="Shirts" data-extra="M" data-price="84"><span class="badge">New</span><h3>Linen shirt</h3><p>Lightweight staple with a clean collar.</p><strong>INR 84.00</strong><button class="primary-button" data-add-to-cart data-name="Linen shirt" data-price="84">Add to cart</button></article><article class="product-card" data-product-card data-name="Tailored blazer" data-category="Outerwear" data-extra="L" data-price="168"><span class="badge">Core</span><h3>Tailored blazer</h3><p>Minimal structure with a sharp lapel.</p><strong>INR 168.00</strong><button class="primary-button" data-add-to-cart data-name="Tailored blazer" data-price="168">Add to cart</button></article><article class="product-card" data-product-card data-name="Cotton dress" data-category="Dresses" data-extra="S" data-price="126"><span class="badge">Seasonal</span><h3>Cotton dress</h3><p>Soft lines with practical movement.</p><strong>INR 126.00</strong><button class="primary-button" data-add-to-cart data-name="Cotton dress" data-price="126">Add to cart</button></article><article class="product-card" data-product-card data-name="Silk scarf" data-category="Accessories" data-extra="S" data-price="52"><span class="badge">Gift</span><h3>Silk scarf</h3><p>Accent layer for evening and travel.</p><strong>INR 52.00</strong><button class="primary-button" data-add-to-cart data-name="Silk scarf" data-price="52">Add to cart</button></article><article class="product-card" data-product-card data-name="Denim jacket" data-category="Outerwear" data-extra="M" data-price="144"><span class="badge">Top seller</span><h3>Denim jacket</h3><p>Structured daily layer with contrast seams.</p><strong>INR 144.00</strong><button class="primary-button" data-add-to-cart data-name="Denim jacket" data-price="144">Add to cart</button></article><article class="product-card" data-product-card data-name="Leather belt" data-category="Accessories" data-extra="L" data-price="46"><span class="badge">Utility</span><h3>Leather belt</h3><p>Minimal finish with matte hardware.</p><strong>INR 46.00</strong><button class="primary-button" data-add-to-cart data-name="Leather belt" data-price="46">Add to cart</button></article><article class="product-card" data-product-card data-name="Wool trouser" data-category="Bottoms" data-extra="M" data-price="132"><span class="badge">Refined</span><h3>Wool trouser</h3><p>Sharp taper for work and travel.</p><strong>INR 132.00</strong><button class="primary-button" data-add-to-cart data-name="Wool trouser" data-price="132">Add to cart</button></article><article class="product-card" data-product-card data-name="Evening top" data-category="Shirts" data-extra="S" data-price="92"><span class="badge">Event</span><h3>Evening top</h3><p>Clean silhouette with subtle sheen.</p><strong>INR 92.00</strong><button class="primary-button" data-add-to-cart data-name="Evening top" data-price="92">Add to cart</button></article></div></div></section>`
	return map[string]string{
		"index.html":       pageHTML(brand+" | Contemporary fashion storefront", "Fashion demo site for SiteCraft.", brand, links, `<section class="hero"><div class="container hero-grid"><div><p class="eyebrow">Fashion store</p><h1>Quiet luxury merchandising with store-style interactions for platform testing.</h1><p class="lead">`+brand+` blends editorial presentation, filterable catalog cards, and a frontend-only cart to test storefront UX inside SiteCraft.</p><a class="primary-button" href="shop.html">Shop the collection</a></div><div class="hero-visual"></div></div></section><section class="section"><div class="container card-grid"><article class="card"><h3>Featured collection</h3><p>Soft tailoring, layered neutrals, and elevated everyday essentials.</p></article><article class="card"><h3>Delivery test</h3><p>Use this site to verify multi-page publish, asset loading, and internal nav.</p></article><article class="card"><h3>Responsive layout</h3><p>Cards stack cleanly on mobile and preserve sharp-edge styling.</p></article></div></section>`, true),
		"about.html":       pageHTML("About | "+brand, "About page for the fashion demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">About</p><h1>Designed as a polished storefront test bed for fashion-focused client demos.</h1><p class="lead">Maison Vela shows how a premium brand can live comfortably in a simple HTML-first workflow.</p></div></section>`, true),
		"shop.html":        pageHTML("Shop | "+brand, "Shop page for the fashion demo.", brand, links, shopBody, true),
		"collections.html": pageHTML("Collections | "+brand, "Collections page for the fashion demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Collections</p><h1>Seasonal groupings that help test secondary pages and navigation flow.</h1></div></section><section class="section"><div class="container card-grid"><article class="card"><h3>Tailored Transit</h3><p>Travel-ready layers, clean outerwear, and versatile trousers.</p></article><article class="card"><h3>Soft Occasion</h3><p>Refined evening pieces with understated detail.</p></article><article class="card"><h3>Daily Uniform</h3><p>Repeatable essentials built around quiet color palettes.</p></article></div></section>`, true),
		"contact.html":     pageHTML("Contact | "+brand, "Contact page for the fashion demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Contact</p><h1>Client service and showroom styling in a compact contact layout.</h1></div></section><section class="section"><div class="container two-column"><form class="contact-form" data-demo-form><input type="text" placeholder="Name" required /><input type="email" placeholder="Email" required /><textarea placeholder="Request styling help, wholesale information, or press details"></textarea><button class="primary-button" type="submit">Send request</button><p class="success-message" data-form-success hidden>Your message was captured in this frontend-only demo state.</p></form><article class="card"><h3>Studio notes</h3><p>Use this page to test form UI and live styling after publish.</p></article></div></section>`, true),
		"styles.css":       siteThemeCSS("#16191d", "#ffffff", "#17191d"),
		"script.js":        defaultStoreScript("maison-vela"),
	}
}

func petSubscriptionWorkspace(brand string) map[string]string {
	links := []demoNavLink{{"Home", "index.html"}, {"About", "about.html"}, {"Subscriptions", "subscriptions.html"}, {"Shop", "shop.html"}, {"FAQ", "faq.html"}, {"Contact", "contact.html"}}
	return map[string]string{
		"index.html":         pageHTML(brand+" | Pet subscription storefront", "Pet subscription demo site for SiteCraft.", brand, links, `<section class="hero"><div class="container hero-grid"><div><p class="eyebrow">Pet subscription store</p><h1>Monthly pet boxes and trust-first subscription merchandising.</h1><p class="lead">Use `+brand+` to test plan cards, subscription-style CTAs, and compact storefront behavior.</p><a class="primary-button" href="subscriptions.html">View plans</a></div><div class="hero-visual"></div></div></section><section class="section"><div class="container metrics-grid"><article class="metric"><strong>4</strong><span>plan variants</span></article><article class="metric"><strong>96%</strong><span>renewal satisfaction</span></article><article class="metric"><strong>12k</strong><span>boxes simulated annually</span></article></div></section>`, true),
		"about.html":         pageHTML("About | "+brand, "About page for the pet subscription demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">About</p><h1>A warm but structured subscription demo with plan-focused journeys.</h1></div></section>`, true),
		"subscriptions.html": pageHTML("Subscriptions | "+brand, "Subscription page for the pet demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Subscriptions</p><h1>Plan cards, selectors, and frontend-only subscribe interactions.</h1></div></section><section class="section"><div class="container"><div class="filter-row"><select data-filter-select data-filter-extra><option value="">All pet types</option><option value="Dog">Dog</option><option value="Cat">Cat</option></select><select data-filter-select data-filter-category><option value="">All frequencies</option><option value="Monthly">Monthly</option><option value="Bi-monthly">Bi-monthly</option></select><select data-filter-select data-filter-sort><option value="">Sort</option><option value="low">Lowest price</option><option value="high">Highest price</option></select><div></div></div><div class="pricing-grid" data-product-grid><article class="plan-card" data-product-card data-name="Puppy Starter Box" data-category="Monthly" data-extra="Dog" data-price="39"><span class="badge">Dog</span><h3>Puppy Starter Box</h3><p>Chews, training rewards, and early enrichment tools.</p><strong>INR 39.00 / month</strong><button class="primary-button" data-add-to-cart data-name="Puppy Starter Box" data-price="39">Subscribe</button></article><article class="plan-card" data-product-card data-name="Cat Comfort Box" data-category="Monthly" data-extra="Cat" data-price="36"><span class="badge">Cat</span><h3>Cat Comfort Box</h3><p>Scratch-friendly toys, comfort textures, and treats.</p><strong>INR 36.00 / month</strong><button class="primary-button" data-add-to-cart data-name="Cat Comfort Box" data-price="36">Subscribe</button></article><article class="plan-card" data-product-card data-name="Healthy Treats Monthly" data-category="Monthly" data-extra="Dog" data-price="28"><span class="badge">Dog</span><h3>Healthy Treats Monthly</h3><p>Rotating snack box for wellness-focused pet owners.</p><strong>INR 28.00 / month</strong><button class="primary-button" data-add-to-cart data-name="Healthy Treats Monthly" data-price="28">Subscribe</button></article><article class="plan-card" data-product-card data-name="Premium Grooming Box" data-category="Bi-monthly" data-extra="Dog" data-price="54"><span class="badge">Care</span><h3>Premium Grooming Box</h3><p>Brushes, wipes, and care products for home routines.</p><strong>INR 54.00 / bi-monthly</strong><button class="primary-button" data-add-to-cart data-name="Premium Grooming Box" data-price="54">Subscribe</button></article></div></div></section>`, true),
		"shop.html":          pageHTML("Shop | "+brand, "Shop page for the pet subscription demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Shop</p><h1>Add-on pet products to test store-style pages alongside subscriptions.</h1></div></section><section class="section"><div class="container card-grid" data-product-grid><article class="product-card" data-product-card data-name="Travel bowl" data-category="Accessories" data-extra="Dog" data-price="18"><h3>Travel bowl</h3><p>Fold-flat hydration companion.</p><strong>INR 18.00</strong><button class="primary-button" data-add-to-cart data-name="Travel bowl" data-price="18">Add to cart</button></article><article class="product-card" data-product-card data-name="Calming toy" data-category="Toys" data-extra="Cat" data-price="16"><h3>Calming toy</h3><p>Soft sensory enrichment for indoor cats.</p><strong>INR 16.00</strong><button class="primary-button" data-add-to-cart data-name="Calming toy" data-price="16">Add to cart</button></article><article class="product-card" data-product-card data-name="Treat pouch" data-category="Accessories" data-extra="Dog" data-price="14"><h3>Treat pouch</h3><p>Daily walk essential with quick access clip.</p><strong>INR 14.00</strong><button class="primary-button" data-add-to-cart data-name="Treat pouch" data-price="14">Add to cart</button></article></div></div></section>`, true),
		"faq.html":           pageHTML("FAQ | "+brand, "FAQ page for the pet subscription demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">FAQ</p><h1>Answers that help test secondary navigation and content-heavy layouts.</h1></div></section><section class="section"><div class="container"><article class="faq-item"><h3>Can owners pause a plan?</h3><p>Yes. This is a demo answer used for presentation and QA.</p></article><article class="faq-item"><h3>How are boxes tailored?</h3><p>By pet type, age, and preference profile in the real concept.</p></article><article class="faq-item"><h3>Is checkout active?</h3><p>No. Subscription and cart interactions are frontend-only in this test site.</p></article></div></section>`, true),
		"contact.html":       pageHTML("Contact | "+brand, "Contact page for the pet subscription demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Contact</p><h1>Support and partnership inquiries.</h1></div></section><section class="section"><div class="container two-column"><form class="contact-form" data-demo-form><input type="text" placeholder="Name" required /><input type="email" placeholder="Email" required /><textarea placeholder="Ask about plans, gifting, or retail partnerships"></textarea><button class="primary-button" type="submit">Contact PawNest Club</button><p class="success-message" data-form-success hidden>Your request was saved in this frontend-only demo.</p></form><article class="card"><h3>Trust badges</h3><p>Vet reviewed selections, flexible skips, and clear box frequency options.</p></article></div></section>`, true),
		"styles.css":         siteThemeCSS("#db6a2b", "#ffffff", "#17212b"),
		"script.js":          defaultStoreScript("pawnest-club"),
	}
}

func customProductsWorkspace(brand string) map[string]string {
	links := []demoNavLink{{"Home", "index.html"}, {"About", "about.html"}, {"Products", "products.html"}, {"Customize", "customize.html"}, {"Pricing", "pricing.html"}, {"Contact", "contact.html"}}
	return map[string]string{
		"index.html":     pageHTML(brand+" | Custom product studio", "Custom products demo site for SiteCraft.", brand, links, `<section class="hero"><div class="container hero-grid"><div><p class="eyebrow">Custom products store</p><h1>Merchandise, kits, and branded collateral with quote-style interactions.</h1><p class="lead">`+brand+` is built to test product pages, custom form UIs, and frontend-only quote/cart actions.</p><a class="primary-button" href="customize.html">Start customization</a></div><div class="hero-visual"></div></div></section>`, true),
		"about.html":     pageHTML("About | "+brand, "About page for the custom products demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">About</p><h1>A compact demo for branded merchandise workflows.</h1></div></section>`, true),
		"products.html":  pageHTML("Products | "+brand, "Products page for the custom products demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Products</p><h1>Custom t-shirts, mugs, totes, stickers, kits, and event merchandise.</h1></div></section><section class="section"><div class="container card-grid" data-product-grid><article class="product-card" data-product-card data-name="Custom t-shirts" data-category="Apparel" data-extra="Small batch" data-price="12"><h3>Custom t-shirts</h3><p>Soft cotton base with flexible print placements.</p><strong>INR 12.00</strong><button class="primary-button" data-add-to-cart data-name="Custom t-shirts" data-price="12">Add to quote</button></article><article class="product-card" data-product-card data-name="Mugs" data-category="Drinkware" data-extra="Corporate" data-price="9"><h3>Mugs</h3><p>Desk-ready ceramic for events and teams.</p><strong>INR 9.00</strong><button class="primary-button" data-add-to-cart data-name="Mugs" data-price="9">Add to quote</button></article><article class="product-card" data-product-card data-name="Corporate kits" data-category="Kits" data-extra="Corporate" data-price="48"><h3>Corporate kits</h3><p>Welcome boxes, launch kits, and partner sends.</p><strong>INR 48.00</strong><button class="primary-button" data-add-to-cart data-name="Corporate kits" data-price="48">Add to quote</button></article></div></div></section>`, true),
		"customize.html": pageHTML("Customize | "+brand, "Customization page for the custom products demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Customize</p><h1>Frontend-only customization form for real workflow testing.</h1></div></section><section class="section"><div class="container two-column"><form class="customizer" data-demo-form><select><option>Custom t-shirts</option><option>Mugs</option><option>Tote bags</option><option>Stickers</option></select><input type="number" min="10" value="50" placeholder="Quantity" /><input type="text" placeholder="Upload design placeholder URL" /><select><option>1 color print</option><option>2 color print</option><option>Full color</option></select><button class="primary-button" type="submit">Request estimate</button><p class="success-message" data-form-success hidden>Estimate requested. This is a frontend-only confirmation state.</p></form><article class="card"><h3>Estimate guide</h3><p>Use this page to test forms, responsive columns, and save/publish behavior across a quote-driven store.</p></article></div></section>`, true),
		"pricing.html":   pageHTML("Pricing | "+brand, "Pricing page for the custom products demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Pricing</p><h1>Tiered pricing to test quote-style card layouts and tables.</h1></div></section><section class="section"><div class="container pricing-grid"><article class="price-card"><h3>Small batch</h3><strong>From INR 9</strong><p>Best for low-volume launches.</p></article><article class="price-card"><h3>Team rollout</h3><strong>From INR 18</strong><p>Balanced for internal kits and events.</p></article><article class="price-card"><h3>Campaign scale</h3><strong>Custom quote</strong><p>For multi-item sets and broad distribution.</p></article></div></section>`, true),
		"contact.html":   pageHTML("Contact | "+brand, "Contact page for the custom products demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Contact</p><h1>Get a quote for your next branded rollout.</h1></div></section><section class="section"><div class="container two-column"><form class="contact-form" data-demo-form><input type="text" placeholder="Company" required /><input type="email" placeholder="Email" required /><textarea placeholder="What are you making and when do you need it?"></textarea><button class="primary-button" type="submit">Send inquiry</button><p class="success-message" data-form-success hidden>Your inquiry was captured in this frontend-only demo state.</p></form><article class="card"><h3>Best fit</h3><p>Ideal for merchandise shops, B2B onboarding kits, and event collateral demos.</p></article></div></section>`, true),
		"styles.css":     siteThemeCSS("#b84518", "#ffffff", "#18181b"),
		"script.js":      defaultStoreScript("printforge-studio"),
	}
}

func digitalProductsWorkspace(brand string) map[string]string {
	links := []demoNavLink{{"Home", "index.html"}, {"About", "about.html"}, {"Products", "products.html"}, {"Downloads", "downloads.html"}, {"Pricing", "pricing.html"}, {"Contact", "contact.html"}}
	return map[string]string{
		"index.html":     pageHTML(brand+" | Digital products storefront", "Digital products demo site for SiteCraft.", brand, links, `<section class="hero"><div class="container hero-grid"><div><p class="eyebrow">Digital products store</p><h1>Templates, kits, and downloadable assets in a frontend-only storefront demo.</h1><p class="lead">`+brand+` exercises product cards, filters, cart behavior, and clean multi-page navigation for digital goods.</p><a class="primary-button" href="products.html">Browse products</a></div><div class="hero-visual"></div></div></section>`, true),
		"about.html":     pageHTML("About | "+brand, "About page for the digital products demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">About</p><h1>A polished storefront for digital creators and lightweight asset shops.</h1></div></section>`, true),
		"products.html":  pageHTML("Products | "+brand, "Products page for the digital products demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Products</p><h1>Searchable templates, packs, and creator tools.</h1></div></section><section class="section"><div class="container"><div class="filter-row"><input data-filter-input type="search" placeholder="Search products" /><select data-filter-select data-filter-category><option value="">All categories</option><option value="Templates">Templates</option><option value="Assets">Assets</option><option value="Guides">Guides</option></select><select data-filter-select data-filter-extra><option value="">All licenses</option><option value="Personal">Personal</option><option value="Commercial">Commercial</option></select><select data-filter-select data-filter-sort><option value="">Sort</option><option value="low">Lowest price</option><option value="high">Highest price</option></select></div><div class="card-grid" data-product-grid><article class="product-card" data-product-card data-name="UI kit" data-category="Assets" data-extra="Commercial" data-price="42"><h3>UI kit</h3><p>Modern interface blocks for dashboards and apps.</p><strong>INR 42.00</strong><button class="primary-button" data-add-to-cart data-name="UI kit" data-price="42">Add to cart</button></article><article class="product-card" data-product-card data-name="Notion template" data-category="Templates" data-extra="Personal" data-price="18"><h3>Notion template</h3><p>Operating system for small teams and creators.</p><strong>INR 18.00</strong><button class="primary-button" data-add-to-cart data-name="Notion template" data-price="18">Add to cart</button></article><article class="product-card" data-product-card data-name="Icon pack" data-category="Assets" data-extra="Commercial" data-price="26"><h3>Icon pack</h3><p>Sharp line icons for websites and product UI.</p><strong>INR 26.00</strong><button class="primary-button" data-add-to-cart data-name="Icon pack" data-price="26">Add to cart</button></article><article class="product-card" data-product-card data-name="Website template" data-category="Templates" data-extra="Commercial" data-price="58"><h3>Website template</h3><p>Launch-ready static site base for agencies.</p><strong>INR 58.00</strong><button class="primary-button" data-add-to-cart data-name="Website template" data-price="58">Add to cart</button></article><article class="product-card" data-product-card data-name="Ebook" data-category="Guides" data-extra="Personal" data-price="14"><h3>Ebook</h3><p>Growth systems guide for service operators.</p><strong>INR 14.00</strong><button class="primary-button" data-add-to-cart data-name="Ebook" data-price="14">Add to cart</button></article><article class="product-card" data-product-card data-name="Analytics dashboard template" data-category="Templates" data-extra="Commercial" data-price="36"><h3>Analytics dashboard template</h3><p>Executive reporting structure for teams and consultants.</p><strong>INR 36.00</strong><button class="primary-button" data-add-to-cart data-name="Analytics dashboard template" data-price="36">Add to cart</button></article></div></div></section>`, true),
		"downloads.html": pageHTML("Downloads | "+brand, "Downloads page for the digital products demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Downloads</p><h1>Instant-download style page with placeholder delivery states.</h1></div></section><section class="section"><div class="container two-column"><article class="card"><h3>Delivery states</h3><p>Use this page to test post-purchase style layouts without backend billing.</p></article><article class="card"><h3>License notes</h3><p>Personal, team, and commercial licenses can be presented cleanly with static content.</p></article></div></section>`, true),
		"pricing.html":   pageHTML("Pricing | "+brand, "Pricing page for the digital products demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Pricing</p><h1>License-based pricing cards for template and asset stores.</h1></div></section><section class="section"><div class="container pricing-grid"><article class="price-card"><h3>Personal</h3><strong>INR 18+</strong><p>Single creator or portfolio use.</p></article><article class="price-card"><h3>Commercial</h3><strong>INR 42+</strong><p>Client work and production-ready deployment.</p></article><article class="price-card"><h3>Studio</h3><strong>Custom</strong><p>Multi-seat access for teams shipping at scale.</p></article></div></section>`, true),
		"contact.html":   pageHTML("Contact | "+brand, "Contact page for the digital products demo.", brand, links, `<section class="page-hero"><div class="container"><p class="eyebrow">Contact</p><h1>Questions about licensing, bundles, or partnerships.</h1></div></section><section class="section"><div class="container two-column"><form class="contact-form" data-demo-form><input type="text" placeholder="Name" required /><input type="email" placeholder="Email" required /><textarea placeholder="Tell us what you want to license or bundle"></textarea><button class="primary-button" type="submit">Send message</button><p class="success-message" data-form-success hidden>Frontend-only success state triggered.</p></form><article class="card"><h3>Use cases</h3><p>Great for testing digital storefront flows, product listing density, and CSS/JS delivery.</p></article></div></section>`, true),
		"styles.css":     siteThemeCSS("#5b46e5", "#ffffff", "#181926"),
		"script.js":      defaultStoreScript("pixelshelf"),
	}
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
	if !cfg.IsDevelopment() {
		return fmt.Errorf("seed is only allowed in development")
	}
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
