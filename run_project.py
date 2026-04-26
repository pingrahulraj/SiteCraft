#!/usr/bin/env python3
from __future__ import annotations

import argparse
import shutil
import json
import os
import platform
import signal
import socket
import subprocess
import sys
import threading
import time
import uuid
from pathlib import Path
from typing import Any
from urllib import parse, request

import psycopg


ROOT = Path(__file__).resolve().parent
BACKEND = ROOT / "backend"
FRONTEND = ROOT / "frontend"
STORAGE = ROOT / "storage"
LOGS = STORAGE / "logs"
ENV_FILE = ROOT / ".env"
PHASE_REPORT = ROOT / "audit" / "phase1_verification_latest.md"
SEED_FILE = STORAGE / "seed_state.json"


MIGRATIONS: list[tuple[str, str]] = [
    (
        "20260425_001_core",
        """
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
create index if not exists idx_workspace_files_site_id on workspace_files(site_id);
create index if not exists idx_site_releases_site_id on site_releases(site_id);
create index if not exists idx_assets_site_id on assets(site_id);
create index if not exists idx_audit_logs_tenant_id on audit_logs(tenant_id);
create index if not exists idx_audit_logs_site_id on audit_logs(site_id);
""",
    ),
    (
        "20260425_002_modules",
        """
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
create index if not exists idx_products_site_id on products(site_id);

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
create index if not exists idx_orders_site_id on orders(site_id);

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
create index if not exists idx_customers_site_id on customers(site_id);

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
create index if not exists idx_coupons_site_id on coupons(site_id);

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
create index if not exists idx_shipping_zones_site_id on shipping_zones(site_id);

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
create index if not exists idx_blog_posts_site_id on blog_posts(site_id);

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
create index if not exists idx_blog_categories_site_id on blog_categories(site_id);

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
create index if not exists idx_blog_tags_site_id on blog_tags(site_id);

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
create index if not exists idx_blog_authors_site_id on blog_authors(site_id);

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
create index if not exists idx_blog_members_site_id on blog_members(site_id);

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
create index if not exists idx_blog_analytics_site_id on blog_analytics(site_id);

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
create index if not exists idx_subscription_plans_site_id on subscription_plans(site_id);

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
create index if not exists idx_subscription_subscribers_site_id on subscription_subscribers(site_id);

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
create index if not exists idx_subscriptions_site_id on subscriptions(site_id);

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
create index if not exists idx_subscription_invoices_site_id on subscription_invoices(site_id);

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
create index if not exists idx_failed_payments_site_id on failed_payments(site_id);

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
""",
    ),
]

REQUIRED_TABLES = [
    "schema_migrations",
    "tenants",
    "users",
    "tenant_users",
    "sites",
    "workspace_files",
    "site_releases",
    "products",
    "blog_posts",
    "subscription_plans",
]

DRIFT_CHECKS = {
    "tenants": {"status", "updated_at"},
    "sites": {"tenant_id", "slug", "status"},
    "workspace_files": {"tenant_id", "site_id", "path"},
    "products": {"tenant_id", "site_id", "slug"},
    "blog_posts": {"tenant_id", "site_id", "slug"},
    "subscription_plans": {"tenant_id", "site_id", "billing_interval"},
}


def load_env() -> dict[str, str]:
    values = dict(os.environ)
    if ENV_FILE.exists():
        for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            values[key.strip()] = value.strip()
    return values


def ensure_dirs() -> None:
    for path in [STORAGE, LOGS, STORAGE / "sites", ROOT / "audit"]:
        path.mkdir(parents=True, exist_ok=True)


def command_env() -> dict[str, str]:
    env = load_env()
    env["STORAGE_ROOT"] = env.get("STORAGE_ROOT", "./storage")
    env["GOCACHE"] = str((STORAGE / ".gocache").resolve())
    return env


def env_mode(env: dict[str, str]) -> str:
    return (env.get("APP_ENV") or env.get("ENV") or "development").lower()


def database_url(env: dict[str, str], admin: bool = False) -> str:
    if env.get("DATABASE_URL") and not admin:
        return env["DATABASE_URL"]
    user = env.get("DB_USER", "postgres")
    password = env.get("DB_PASSWORD", "")
    host = env.get("DB_HOST", "127.0.0.1")
    port = env.get("DB_PORT", "5432")
    name = env.get("DB_ADMIN_DB", "postgres") if admin else env.get("DB_NAME", "websitebuilder")
    sslmode = env.get("DB_SSLMODE", "disable")
    return f"postgresql://{parse.quote(user)}:{parse.quote(password)}@{host}:{port}/{name}?sslmode={sslmode}"


def connect_db(env: dict[str, str], admin: bool = False, autocommit: bool = False):
    conn = psycopg.connect(database_url(env, admin=admin), autocommit=autocommit)
    return conn


def append_log(name: str, message: str) -> None:
    ensure_dirs()
    with (LOGS / name).open("a", encoding="utf-8") as handle:
        handle.write(message.rstrip() + "\n")


def run(command: list[str], cwd: Path, log_name: str, env: dict[str, str] | None = None) -> int:
    exec_env = env or command_env()
    ensure_dirs()
    with (LOGS / log_name).open("a", encoding="utf-8") as output:
        result = subprocess.run(command, cwd=cwd, env=exec_env, stdout=output, stderr=subprocess.STDOUT, check=False)
    return result.returncode


def deterministic_uuid(seed: str) -> str:
    return str(uuid.uuid5(uuid.NAMESPACE_OID, seed))


def site_workspace_dir(tenant_id: str, site_id: str) -> Path:
    return STORAGE / "sites" / tenant_id / site_id / "workspace"


def site_published_dir(tenant_id: str, site_id: str) -> Path:
    return STORAGE / "sites" / tenant_id / site_id / "published"


def default_html(name: str) -> str:
    return f"""<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>{name}</title>
  </head>
  <body>
    <main class="page">
      <h1>{name}</h1>
    </main>
  </body>
</html>
"""


def sync_workspace_tree(conn, tenant_id: str, site_id: str, workspace: Path, published: Path) -> bool:
    existing_files = [path for path in workspace.rglob("*") if path.is_file()]
    if not existing_files:
        return False
    for path in existing_files:
        relative = path.relative_to(workspace)
        target = published / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(path, target)
        sync_workspace_metadata(
            conn,
            tenant_id,
            site_id,
            relative.as_posix(),
            relative.suffix.lstrip(".") or "txt",
            path.stat().st_size,
        )
    return True


def create_db() -> int:
    env = command_env()
    try:
        with connect_db(env, admin=True, autocommit=True) as conn:
            with conn.cursor() as cur:
                cur.execute("select exists(select 1 from pg_database where datname = %s)", (env["DB_NAME"],))
                exists = cur.fetchone()[0]
                if not exists:
                    cur.execute(f'create database "{env["DB_NAME"].replace(chr(34), chr(34) * 2)}"')
        append_log("api.log", "database create-db completed")
        return 0
    except Exception as exc:  # noqa: BLE001
        append_log("api.log", f"create-db failed: {exc}")
        return 1


def detect_schema_drift(conn) -> list[str]:
    drifts: list[str] = []
    with conn.cursor() as cur:
        for table, columns in DRIFT_CHECKS.items():
            cur.execute(
                """
                select column_name from information_schema.columns
                where table_schema = 'public' and table_name = %s
                """,
                (table,),
            )
            present = {row[0] for row in cur.fetchall()}
            if present and not columns.issubset(present):
                drifts.append(f"{table}: missing {', '.join(sorted(columns - present))}")
    return drifts


def ensure_schema_migrations(conn) -> None:
    with conn.cursor() as cur:
        cur.execute(
            """
            create table if not exists schema_migrations (
                version text primary key,
                applied_at timestamptz not null default now()
            )
            """
        )
    conn.commit()


def migrate_conn(conn) -> None:
    ensure_schema_migrations(conn)
    with conn.cursor() as cur:
        for version, sql_text in MIGRATIONS:
            cur.execute("select exists(select 1 from schema_migrations where version = %s)", (version,))
            if cur.fetchone()[0]:
                continue
            cur.execute(sql_text)
            cur.execute("insert into schema_migrations(version) values (%s)", (version,))
    conn.commit()


def sync_workspace_metadata(conn, tenant_id: str, site_id: str, path: str, file_type: str, size_bytes: int) -> None:
    with conn.cursor() as cur:
        cur.execute(
            """
            insert into workspace_files(id, tenant_id, site_id, path, file_type, size_bytes, checksum, created_at, updated_at)
            values (%s, %s, %s, %s, %s, %s, null, now(), now())
            on conflict (site_id, path)
            do update set file_type = excluded.file_type, size_bytes = excluded.size_bytes, updated_at = now()
            """,
            (str(uuid.uuid4()), tenant_id, site_id, path, file_type, size_bytes),
        )
    conn.commit()


def seed(conn, env: dict[str, str]) -> None:
    tenant_id = deterministic_uuid("tenant-demo")
    user_id = deterministic_uuid("user-demo")
    site_id = deterministic_uuid("site-demo")
    tenant_user_id = deterministic_uuid("tenant-user-demo")
    with conn.cursor() as cur:
        cur.execute(
            """
            insert into tenants(id, name, slug, status, created_at, updated_at)
            values (%s, %s, %s, 'active', now(), now())
            on conflict (slug) do update set name = excluded.name, updated_at = now()
            """,
            (tenant_id, "Acme Studio", "acme-studio"),
        )
        cur.execute(
            """
            insert into users(id, email, name, password_hash, status, created_at, updated_at)
            values (%s, %s, %s, %s, 'active', now(), now())
            on conflict (email) do update set name = excluded.name, password_hash = excluded.password_hash, updated_at = now()
            """,
            (user_id, "ava@acme-studio.com", "Ava Chen", "$2a$10$jOpJFE5vHwmL/x/6Mi9I3OUE9rNQKtc8eYI2bMVIlIyQOqSUkGIzK"),
        )
        cur.execute(
            """
            insert into tenant_users(id, tenant_id, user_id, role, status, created_at, updated_at)
            values (%s, %s, %s, 'owner', 'active', now(), now())
            on conflict (tenant_id, user_id) do update set role = 'owner', status = 'active', updated_at = now()
            """,
            (tenant_user_id, tenant_id, user_id),
        )
        cur.execute(
            """
            insert into sites(id, tenant_id, name, slug, status, created_at, updated_at)
            values (%s, %s, %s, %s, 'draft', now(), now())
            on conflict (tenant_id, slug) do update set name = excluded.name, updated_at = now()
            """,
            (site_id, tenant_id, "Client landing page", "client-landing-page"),
        )
        cur.execute(
            """
            insert into products(id, tenant_id, site_id, name, slug, description, price, sku, category, stock_status, status, created_at, updated_at)
            values (%s, %s, %s, 'Starter kit', 'starter-kit', 'Starter product for the site storefront.', 49, 'SC-001', 'General', 'in_stock', 'active', now(), now())
            on conflict (site_id, slug) do nothing
            """,
            (deterministic_uuid("product-starter"), tenant_id, site_id),
        )
        cur.execute(
            """
            insert into blog_posts(id, tenant_id, site_id, title, slug, excerpt, content, author, tags_json, seo_title, seo_description, visibility, status, published_at, created_at, updated_at)
            values (%s, %s, %s, 'Launching the new site', 'launching-the-new-site', 'A short update on the new site rollout.', '<p>We are rolling out the new site experience.</p>', 'Ava Chen', '["launch","product"]'::jsonb, 'Launching the new site', 'A short update on the new site rollout.', 'public', 'published', now(), now(), now())
            on conflict (site_id, slug) do nothing
            """,
            (deterministic_uuid("post-launch"), tenant_id, site_id),
        )
        cur.execute(
            """
            insert into subscription_plans(id, tenant_id, site_id, name, description, price, billing_interval, trial_days, status, created_at, updated_at)
            values (%s, %s, %s, 'Essentials box', 'Monthly curation for repeat customers.', 29, 'monthly', 7, 'active', now(), now())
            on conflict do nothing
            """,
            (deterministic_uuid("plan-essentials"), tenant_id, site_id),
        )
    conn.commit()

    workspace = site_workspace_dir(tenant_id, site_id)
    published = site_published_dir(tenant_id, site_id)
    workspace.mkdir(parents=True, exist_ok=True)
    published.mkdir(parents=True, exist_ok=True)
    if not sync_workspace_tree(conn, tenant_id, site_id, workspace, published):
        files = {
            "index.html": default_html("Client landing page"),
            "styles.css": "body { margin: 0; font-family: Arial, sans-serif; background: #f4f7fb; }\n.page { padding: 48px; }\n",
            "app.js": "console.log('WebsiteBuilder ready');\n",
        }
        for name, content in files.items():
            (workspace / name).write_text(content, encoding="utf-8")
            (published / name).write_text(content, encoding="utf-8")
            sync_workspace_metadata(conn, tenant_id, site_id, name, name.split(".")[-1], len(content.encode("utf-8")))

    with conn.cursor() as cur:
        cur.execute("select coalesce(max(release_number), 0) from site_releases where site_id = %s", (site_id,))
        release_number = cur.fetchone()[0] + 1
        cur.execute(
            """
            insert into site_releases(id, tenant_id, site_id, release_number, artifact_path, public_url, status, created_at)
            values (%s, %s, %s, %s, %s, %s, 'published', now())
            on conflict (site_id, release_number) do nothing
            """,
            (
                str(uuid.uuid4()),
                tenant_id,
                site_id,
                release_number,
                f"sites/{tenant_id}/{site_id}/published",
                f"{delivery_base_url(env)}/sites/{site_id}/client-landing-page",
            ),
        )
        cur.execute(
            "update sites set status = 'published', published_at = now(), updated_at = now() where id = %s",
            (site_id,),
        )
    conn.commit()
    SEED_FILE.write_text(json.dumps({"tenantId": tenant_id, "userId": user_id, "siteId": site_id}, indent=2), encoding="utf-8")


def verify_seed(conn) -> list[str]:
    failures: list[str] = []
    with conn.cursor() as cur:
        cur.execute("select count(*) from tenants where slug = 'acme-studio'")
        if cur.fetchone()[0] != 1:
            failures.append("demo tenant missing")
        cur.execute("select count(*) from users where email = 'ava@acme-studio.com'")
        if cur.fetchone()[0] != 1:
            failures.append("demo user missing")
        cur.execute("select count(*) from tenant_users")
        if cur.fetchone()[0] < 1:
            failures.append("tenant_user missing")
        cur.execute("select count(*) from sites where slug = 'client-landing-page'")
        if cur.fetchone()[0] != 1:
            failures.append("demo site missing")
    if not SEED_FILE.exists():
        failures.append("seed_state.json missing")
    else:
        state = json.loads(SEED_FILE.read_text(encoding="utf-8"))
        if not (site_workspace_dir(state["tenantId"], state["siteId"]) / "index.html").exists():
            failures.append("workspace index.html missing")
    return failures


def init() -> int:
    env = command_env()
    ensure_dirs()
    if create_db() != 0:
        return 1
    try:
        with connect_db(env) as conn:
            drifts = detect_schema_drift(conn)
            if drifts:
                message = "schema drift detected; run: python run_project.py reset-db"
                append_log("api.log", message + " :: " + "; ".join(drifts))
                print(message)
                return 1
            migrate_conn(conn)
            if env_mode(env) in {"development", "dev"}:
                seed(conn, env)
                failures = verify_seed(conn)
                if failures:
                    append_log("api.log", "seed verification failed: " + "; ".join(failures))
                    return 1
        append_log("api.log", "init completed successfully")
        return 0
    except Exception as exc:  # noqa: BLE001
        append_log("api.log", f"init failed: {exc}")
        return 1


def reset_db() -> int:
    env = command_env()
    if env_mode(env) not in {"development", "dev"}:
        print("reset-db is only allowed in development")
        return 1
    try:
        with connect_db(env, admin=True, autocommit=True) as conn:
            with conn.cursor() as cur:
                cur.execute(
                    """
                    select pg_terminate_backend(pid)
                    from pg_stat_activity
                    where datname = %s and pid <> pg_backend_pid()
                    """,
                    (env["DB_NAME"],),
                )
                cur.execute(f'drop database if exists "{env["DB_NAME"].replace(chr(34), chr(34) * 2)}"')
                cur.execute(f'create database "{env["DB_NAME"].replace(chr(34), chr(34) * 2)}"')
        with connect_db(env) as conn:
            migrate_conn(conn)
            seed(conn, env)
            failures = verify_seed(conn)
            if failures:
                append_log("api.log", "reset-db verification failed: " + "; ".join(failures))
                return 1
        append_log("api.log", "reset-db completed successfully")
        print("development database reset and seeded")
        return 0
    except Exception as exc:  # noqa: BLE001
        append_log("api.log", f"reset-db failed: {exc}")
        return 1


def delivery_base_url(env: dict[str, str]) -> str:
    return env.get("DELIVERY_BASE_URL") or f"http://localhost:{env.get('DELIVERY_PORT', '8083')}"


def service_ready(url: str, timeout: float = 1.0) -> bool:
    try:
        with request.urlopen(url, timeout=timeout) as response:
            return response.status == 200
    except Exception:  # noqa: BLE001
        return False


def test() -> int:
    env = command_env()
    ensure_dirs()
    failures = []
    if run([sys.executable, "tests/run_all.py"], ROOT, "test.log", env) != 0:
        failures.append("python smoke tests")
    go_rc = run(["go", "test", "./..."], BACKEND, "test.log", env)
    if go_rc != 0:
        append_log("test.log", "warning: go test could not complete cleanly")
    npm_rc = run(["npm.cmd" if os.name == "nt" else "npm", "run", "build"], FRONTEND, "test.log", env)
    if npm_rc != 0:
        append_log("test.log", "warning: frontend build could not complete cleanly from runner")
    return 1 if failures else 0


def clear() -> None:
    os.system("cls" if os.name == "nt" else "clear")


def fmt_status(proc: subprocess.Popen | None) -> str:
    if proc is None:
        return "stopped"
    return "running" if proc.poll() is None else f"exited({proc.returncode})"


def render(backend_proc: subprocess.Popen | None, frontend_proc: subprocess.Popen | None, env: dict[str, str]) -> None:
    clear()
    print("=" * 84)
    print("WEBSITEBUILDER DEV WORKSPACE")
    print("=" * 84)
    print(f"Host OS        : {platform.system()} {platform.release()}")
    print(f"Python         : {platform.python_version()}")
    print(f"Repository     : {ROOT}")
    print(f"Storage        : {STORAGE}")
    print(f"Logs           : {LOGS}")
    print("")
    print("Services")
    print(f"  API          : {fmt_status(backend_proc)} -> http://localhost:{env.get('API_PORT', '8080')}/api")
    print(f"  Frontend     : {fmt_status(frontend_proc)} -> http://localhost:{env.get('FRONTEND_PORT', '5137')}")
    print(f"  Delivery     : {fmt_status(backend_proc)} -> {delivery_base_url(env)}/sites/{json.loads(SEED_FILE.read_text(encoding='utf-8')).get('siteId', 'site-id')}/client-landing-page" if SEED_FILE.exists() else f"  Delivery     : {fmt_status(backend_proc)} -> {delivery_base_url(env)}/sites/<site_id>/client-landing-page")
    print("")
    print("Commands")
    print("  q + Enter    : quit")
    print("  Ctrl+C       : quit")
    print("=" * 84)


def spawn(command: list[str], cwd: Path, log_name: str, env: dict[str, str]) -> subprocess.Popen:
    handle = (LOGS / log_name).open("a", encoding="utf-8")
    return subprocess.Popen(command, cwd=cwd, env=env, stdout=handle, stderr=subprocess.STDOUT)


def shutdown(procs: list[subprocess.Popen]) -> None:
    for proc in procs:
        if proc.poll() is None:
            proc.terminate()
    time.sleep(1)
    for proc in procs:
        if proc.poll() is None:
            proc.kill()


def run_dev() -> int:
    env = command_env()
    if init() != 0:
        return 1
    npm = "npm.cmd" if os.name == "nt" else "npm"
    frontend_port = env.get("FRONTEND_PORT", "5137")
    backend_proc = spawn(["go", "run", "./cmd/api"], BACKEND, "api.log", env)
    frontend_proc = spawn([npm, "run", "dev", "--", "--host", "127.0.0.1", "--port", frontend_port, "--strictPort"], FRONTEND, "frontend.log", env)
    procs = [backend_proc, frontend_proc]
    stop_requested = threading.Event()
    stop_reason = ""
    failed_log = ""

    def recent_log_lines(path: str, limit: int = 8) -> list[str]:
        log_path = Path(path)
        if not log_path.exists():
            return []
        lines = [line.rstrip() for line in log_path.read_text(encoding="utf-8", errors="ignore").splitlines() if line.strip()]
        return lines[-limit:]

    def watch_input() -> None:
        while not stop_requested.is_set():
            line = sys.stdin.readline()
            if line and line.strip().lower() == "q":
                stop_requested.set()

    try:
        if sys.stdin.isatty():
            threading.Thread(target=watch_input, daemon=True).start()
        while True:
            render(backend_proc, frontend_proc, env)
            if stop_requested.is_set():
                stop_reason = "Stopped by user."
                break
            backend_code = backend_proc.poll()
            frontend_code = frontend_proc.poll()
            if backend_code is not None:
                stop_reason = f"Backend exited with code {backend_code}."
                failed_log = str(LOGS / "api.log")
                break
            if frontend_code is not None:
                stop_reason = f"Frontend exited with code {frontend_code}."
                failed_log = str(LOGS / "frontend.log")
                break
            time.sleep(1)
        return 0 if stop_requested.is_set() else 1
    finally:
        stop_requested.set()
        shutdown(procs)
        clear()
        if stop_reason:
            print(stop_reason)
            if failed_log:
                print(f"See log: {failed_log}")
                recent = recent_log_lines(failed_log)
                if recent:
                    print("Recent log lines:")
                    for line in recent:
                        print(f"  {line}")
        print("WebsiteBuilder stopped.")


def update_phase_report(commands_run: list[str], status: str, blockers: list[str], changed_files: list[str]) -> None:
    ensure_dirs()
    lines = [
        "# Phase 1 Verification",
        "",
        f"- Date: {time.strftime('%Y-%m-%d %H:%M:%S')}",
        "- Commands run:",
    ]
    lines.extend([f"  - `{cmd}`" for cmd in commands_run])
    lines.extend(
        [
            "",
            "## Status",
            f"- Final status: {status}",
            "",
            "## Blockers Found",
        ]
    )
    if blockers:
        lines.extend([f"- {item}" for item in blockers])
    else:
        lines.append("- None")
    lines.extend(["", "## Files Changed"])
    lines.extend([f"- `{item}`" for item in changed_files] if changed_files else ["- None"])
    PHASE_REPORT.write_text("\n".join(lines) + "\n", encoding="utf-8")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="WebsiteBuilder project runner")
    parser.add_argument("command", choices=["init", "create-db", "migrate", "seed", "reset-db", "test", "run-dev"], nargs="?", default="run-dev")
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    env = command_env()
    ensure_dirs()

    if args.command == "create-db":
        return create_db()
    if args.command == "migrate":
        try:
            with connect_db(env) as conn:
                drifts = detect_schema_drift(conn)
                if drifts:
                    print("schema drift detected; run: python run_project.py reset-db")
                    return 1
                migrate_conn(conn)
            return 0
        except Exception as exc:  # noqa: BLE001
            append_log("api.log", f"migrate failed: {exc}")
            return 1
    if args.command == "seed":
        if env_mode(env) not in {"development", "dev"}:
            print("seed is only allowed in development")
            return 1
        try:
            with connect_db(env) as conn:
                seed(conn, env)
                return 0 if not verify_seed(conn) else 1
        except Exception as exc:  # noqa: BLE001
            append_log("api.log", f"seed failed: {exc}")
            return 1
    if args.command == "reset-db":
        return reset_db()
    if args.command == "init":
        return init()
    if args.command == "test":
        return test()
    if args.command == "run-dev":
        return run_dev()
    return 1


if __name__ == "__main__":
    if os.name == "nt":
        signal.signal(signal.SIGINT, signal.default_int_handler)
    raise SystemExit(main())
