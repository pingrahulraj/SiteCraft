from __future__ import annotations

import time
import uuid

from .common import api_base, auth_headers, db_connect, http_json, login_token, service_available


def ensure_second_tenant() -> tuple[str, str]:
    tenant_id = str(uuid.uuid5(uuid.NAMESPACE_OID, "tenant-beta"))
    user_id = str(uuid.uuid5(uuid.NAMESPACE_OID, "user-beta"))
    membership_id = str(uuid.uuid5(uuid.NAMESPACE_OID, "tenant-user-beta"))
    with db_connect() as conn:
        with conn.cursor() as cur:
            cur.execute("select password_hash from users where email = %s", ("ava@acme-studio.com",))
            password_hash = cur.fetchone()[0]
            cur.execute(
                """
                insert into tenants(id, name, slug, status, created_at, updated_at)
                values (%s, %s, %s, 'active', now(), now())
                on conflict (slug) do update set name = excluded.name, updated_at = now()
                """,
                (tenant_id, "Beta Studio", "beta-studio"),
            )
            cur.execute(
                """
                insert into users(id, email, name, password_hash, status, created_at, updated_at)
                values (%s, %s, %s, %s, 'active', now(), now())
                on conflict (email) do update set name = excluded.name, password_hash = excluded.password_hash, updated_at = now()
                """,
                (user_id, "beta@studio.test", "Beta Owner", password_hash),
            )
            cur.execute(
                """
                insert into tenant_users(id, tenant_id, user_id, role, status, created_at, updated_at)
                values (%s, %s, %s, 'owner', 'active', now(), now())
                on conflict (tenant_id, user_id) do update set role = 'owner', status = 'active', updated_at = now()
                """,
                (membership_id, tenant_id, user_id),
            )
        conn.commit()
    return tenant_id, user_id


def run():
    if not service_available(f"{api_base().rstrip('/api')}/healthz"):
        return True

    ensure_second_tenant()
    owner_headers = auth_headers(login_token())
    beta_headers = auth_headers(login_token("beta", "Welcome123!"))

    name = f"Beta Isolation Site {int(time.time())}"
    create_status, create_data = http_json(
        "POST",
        f"{api_base()}/projects",
        {"tenantSlug": "beta-studio", "name": name, "description": "tenant isolation"},
        headers=beta_headers,
    )
    if create_status not in (200, 201):
        return f"beta site create failed: {create_status}"
    with db_connect() as conn:
        with conn.cursor() as cur:
            cur.execute(
                """
                select s.id
                from sites s
                join tenants t on t.id = s.tenant_id
                where t.slug = %s and s.name = %s
                order by s.created_at desc
                limit 1
                """,
                ("beta-studio", name),
            )
            row = cur.fetchone()
    if not row:
        return "beta site missing from database after create"
    beta_site_id = row[0]

    own_status, _ = http_json("GET", f"{api_base()}/projects/{beta_site_id}", headers=beta_headers)
    if own_status != 200:
        return f"beta owner could not read own site: {own_status}"

    cross_read_status, cross_read_data = http_json("GET", f"{api_base()}/projects/{beta_site_id}", headers=owner_headers)
    if cross_read_status != 403 or ((cross_read_data.get("error") or {}).get("code")) != "forbidden":
        return f"cross-tenant site read should be 403, got {cross_read_status}"

    cross_write_status, cross_write_data = http_json(
        "PUT",
        f"{api_base()}/sites/{beta_site_id}/workspace/files/content",
        {"path": "index.html", "content": "<html><body>blocked</body></html>"},
        headers=owner_headers,
    )
    if cross_write_status != 403 or ((cross_write_data.get("error") or {}).get("code")) != "forbidden":
        return f"cross-tenant workspace write should be 403, got {cross_write_status}"

    cross_publish_status, cross_publish_data = http_json(
        "POST",
        f"{api_base()}/sites/{beta_site_id}/publish",
        {},
        headers=owner_headers,
    )
    if cross_publish_status != 403 or ((cross_publish_data.get("error") or {}).get("code")) != "forbidden":
        return f"cross-tenant publish should be 403, got {cross_publish_status}"

    for path in ("products", "posts", "subscription-plans"):
        status, body = http_json("GET", f"{api_base()}/sites/{beta_site_id}/{path}", headers=owner_headers)
        if status != 403 or ((body.get("error") or {}).get("code")) != "forbidden":
            return f"cross-tenant {path} read should be 403, got {status}"

    return True
