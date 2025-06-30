# Security & Access Control

## Authentication

- JWT Bearer authentication is required for all API endpoints except `/api/login`.
- JWT is stored client-side (e.g. in local storage).

## User Roles

- Users are defined in `users.yaml` (if using file backend).
- `admin: true` users have access to reserved dimensions/metrics and admin features.

## Rights Management

- Any dimension or metric with `reserved: true` in `druid.yaml` is admin-only.
- All rights are checked at each report execution request (401/403 responses, detailed logs).

### Row-level security (mandatory filters with `access`)

You can restrict user access to specific rows by configuring the `access` field for each user in `users.yaml`.  
This allows you to enforce mandatory filters on certain dimensions, per datasource.

**Example:**
```yaml
users:
  alice:
    hash: "..."
    salt: "..."
    admin: false
    access:
      myreport:
        browser: ["Chrome", "Firefox"]
        country: ["FR"]
```

With this configuration, all queries executed by `alice` on the datasource `myreport` will automatically include filters:
- `browser` IN `["Chrome", "Firefox"]`
- `country` IN `["FR"]`

This ensures the user only sees data matching these values, regardless of the filters they select in the UI.

## Static Files

- Only files explicitly listed in `static_allowed` are served.
- Wildcards are supported for convenience.

## Logging

- All authentication attempts, API calls, static file accesses, and report executions are logged for audit and troubleshooting.

## Best Practices

- Use strong, unique passwords for all users.
- Restrict static file exposure to only what is needed.
- Regularly review logs and user access.

---