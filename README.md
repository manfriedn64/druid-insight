
# druid-insight

**druid-insight** is an advanced open-source dashboard for Apache Druid®, designed as an open alternative to Turnilo. It features authentication, user/admin rights, asynchronous querying, custom metric formulas, REST API, modular static serving, logging, and more.

---

## **Project structure**

```
druid-insight/
├── cmd/
│   ├── druid-insight/
│   │   └── main.go         # Main HTTP server
│   └── service/
│       └── main.go         # Service manager: start/stop/reload (daemon)
├── api/                    # REST API endpoints (login, schema, execute, status)
├── auth/                   # Authentication, rights, JWT, users
├── druid/                  # Druid mapping, formula parser, native querying
├── worker/                 # FIFO queue, workers, request/result structs
├── static/                 # Secure static file handler
├── logging/                # Loggers (multi-file)
├── config.yaml             # Main configuration
├── druid.yaml              # Datasource/dimension/metric mapping
├── users.yaml              # User list (if backend is "file")
├── go.mod / go.sum         # Go module files
├── bin/                    # Compiled binaries
└── README.md               # This file
```

---

## **Installation**

**Requirements:**
- Go 1.21 or later
- An Apache Druid® cluster deployed somewhere

**Clone the repo and install dependencies:**
```sh
git clone <URL> druid-insight
cd druid-insight
go mod tidy
```

---

## **Build**

```sh
make build      # Build the binaries in bin/
```

- `bin/druid-insight` : the main HTTP server
- `bin/service` : the service manager (start/stop/reload, daemon mode)
- `bin/userctl` : CLI for user management

---

## **Configuration**

- `config.yaml`: server parameters, JWT, user backend, logs, static files
- `druid.yaml`: all datasources, dimensions and metrics mapping (with custom formulas)
- `users.yaml`: users (if backend is "file"), with hash/salt/admin

---

## Example configuration files

**config.yaml**
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

**druid.yaml**
```yaml
host_url: "http://localhost:8082/query"
datasources:
  myreport:
    dimensions:
      date:
        druid: __time
        reserved: false
      regie:
        druid: network
        reserved: true
      ssp:
        druid: ssp
        reserved: false
    metrics:
      revenue:
        druid: revenue
        reserved: false
      imps_sold:
        druid: impressions
        reserved: false
      cpm:
        formula: "1000 * revenue / impressions"
        reserved: true
```

**users.yaml**
```yaml
users:
  admin:
    hash: "abcdef0123456789..."   # SHA256 hash of the password
    salt: "somesalt"
    admin: true
  alice:
    hash: "fedaedcba9876543..."
    salt: "anothersalt"
    admin: false
```
---

## User management (CLI)

A CLI utility is provided to manage users when using the `file` backend (`users.yaml`).

### Build

The command is built along with the rest of the project:

```sh
make build     # or manually: go build -o bin/userctl cmd/userctl/main.go
```

### Usage

```sh
bin/userctl add <username>         # Add a user (password asked, admin role offered)
bin/userctl disable <username>     # Soft-disable: comments the user entry in users.yaml
bin/userctl list                   # Lists all active (non-commented) users
```

- **Example:**

    ```sh
    bin/userctl add alice
    bin/userctl disable alice
    bin/userctl list
    ```

- When disabling, the user entry is commented out in the YAML for easy audit or future reactivation.
- To reactivate a user, simply uncomment their section in `users.yaml`.

---

## **Running the server**

### **As a service (recommended):**

```sh
make start       # Start the daemon in the background (PID in /tmp/druid-insight.pid)
make stop        # Stop the server (SIGTERM)
make reload      # Reload configuration at runtime (SIGHUP)
```
_Internally, `bin/service` manages process launching and signals._

### **Direct mode (for development):**

```sh
make run         # Start the server interactively (CTRL+C to stop)
```

---

## **Usage**

The server exposes:

- `/api/login`: Authentication (POST username/password → JWT)
- `/api/schema`: Full schema: available dimensions/metrics per user/admin (JWT required)
- `/api/report/execute`: Launch an asynchronous report (JWT, JSON payload: datasource/dimensions/metrics/filters)
- `/api/report/status?id=...`: Poll a report’s status (waiting/processing/complete/error), retrieve the result

**Static files (UI, JS, CSS) are served securely via a whitelist, with fallback support for easy theming/modding.**

---

## **Logs**

Three log files (default in `./logs/`):

- `access.log`  — API calls, static file access
- `login.log`   — Authentication successes/failures
- `report.log`  — Report execution, worker logs

---

## **Security & Rights**

- JWT Bearer authentication (JWT stored in client local storage)
- Any `reserved: true` dimension or metric in `druid.yaml` is admin-only
- All rights are checked at each report execution request (401/403 + detailed log)
- Static files are strictly whitelisted (wildcard supported), fallback to admin/static_default

---

## **Custom metric formulas**

- Supports complex arithmetic formulas for metrics (`cpm: 1000 * revenue / impressions`, etc.), translated **directly into Druid postAggregations** (as in Turnilo/Superset).
- Parsing is secured: it is impossible to use undeclared or reserved metrics/dimensions without the proper rights.

---

## **Testing**

Run all unit tests:

```sh
make test
# or
go test ./...
```

**Add your own tests in each folder (`*_test.go`).**

---

## **Extending**

- To add metrics/dimensions, simply update `druid.yaml` (supports formulas and mapping).
- To change rights, edit the `reserved` flag or set users as `admin: true/false`.
- To use a SQL backend for users, set `auth.user_backend` and a SQL query in `config.yaml`.

---

## **Contributing**

Feel free to:

- Open a PR for a bug, an endpoint or a new chart type
- Add tests, CI, documentation
- Discuss project evolution in the GitHub repo

---

## **License**

This project is open source, MIT licensed.  
Apache Druid® is a trademark of Imply Data, Inc.  
This project is not affiliated with Imply or Turnilo.

---

**Need help with configuration, an extension or the code?  
Contact the maintainer or open an issue.**

---
