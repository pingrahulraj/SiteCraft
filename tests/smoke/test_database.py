from __future__ import annotations

from .common import db_connect


def run():
    required = ["tenants", "users", "tenant_users", "sites", "workspace_files", "site_releases", "products", "blog_posts", "subscription_plans"]
    with db_connect() as conn:
        with conn.cursor() as cur:
            for table in required:
                cur.execute("select exists(select 1 from information_schema.tables where table_schema='public' and table_name=%s)", (table,))
                if not cur.fetchone()[0]:
                    return f"missing table {table}"
            cur.execute("select count(*) from tenants where slug='acme-studio'")
            if cur.fetchone()[0] < 1:
                return "demo tenant missing"
            cur.execute("select count(*) from sites where slug='client-landing-page'")
            if cur.fetchone()[0] < 1:
                return "demo site missing"
    return True
