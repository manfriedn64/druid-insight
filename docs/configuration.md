# Configuration

druid-insight uses three main configuration files at the project root:

---

## 1. `config.yaml`

General server, authentication, logging, and static file settings.

```yaml
server:
  listen: ":8080"
  static: "./static"
  static_default: "./static"
  static_allowed:
    - "index.html"
    - "css/style.css"
    - "js/*.js"
  log_dir: "./logs"

jwt:
  secret: "a_super_secret_passphrase"
  expiration_minutes: 120

auth:
  user_backend: "file"
  user_file: "users.yaml"
  hash_macro: "{sha256}({password}{user}{salt}{globalsalt})"
  salt: "mysalt"
```

---

## 2. `druid.yaml`

Datasource, dimension, and metric mapping (with support for custom formulas).

```yaml
host_url: "http://localhost:8082/query"

datasources:
  myreport:
    dimensions:
      date:
        druid: __time
        reserved: false
      browser:
        druid: browser
        reserved: true
      device:
        druid: device
        reserved: false
    metrics:
      errors:
        druid: errors
        reserved: true
      requests:
        druid: requests
        reserved: false
      errorrate:
        formula: "100 * errors / requests"
        reserved: true
```

---

## 3. `users.yaml`

User accounts and roles (used if `auth.user_backend` is `"file"`).

```yaml
users:
  admin:
    hash: "abcdef0123456789..."
    salt: "somesalt"
    admin: true
  alice:
    hash: "fedaedcba9876543..."
    salt: "anothersalt"
    admin: false
```

---

**Tip:**  
To generate a SHA256 hash for a password, you can use the provided `userctl` CLI or a script matching your `hash_macro`.

---

**See also:**  
- [docs/security.md](security.md) for rights management.
- [docs/static.md](static.md) for static file customization.