# druid-insight

**druid-insight** is an open-source web dashboard application for Apache Druid®.  
It allows you to securely query, visualize, and export Druid data with advanced customization.

---

## Features

- **Modern web interface** for building and running analytical reports on Druid.
- **JWT authentication** with user/admin role management.
- **Fine-grained access control**: whitelists for dimensions, metrics, and static files.
- **Asynchronous queries**: FIFO queue, workers, status polling, CSV generation and download.
- **Custom formulas**: support for Druid post-aggregations and advanced formulas.
- **REST API** for report execution, schema access, filter management, and more.
- **Fast and secure CSV exports**.
- **Advanced customization**:
  - Themes and static files can be overridden via an admin folder.
  - Dynamic variable injection into static files (macros `{VAR}`).
  - Automatic fallback to a default folder if a file is not found.
- **Enhanced security**:
  - Static files are only served if present in a whitelist (wildcard support).
  - All access/refusal/status events are logged.
- **Integrated visualizations** (tables, charts via Chart.js).
- **Report sharing** via direct links.
- **Simple deployment** (Go binary, single configuration).

---

## Installation

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

## Build

```sh
make build      # Build the binaries in bin/
```

- `bin/druid-insight` : the main HTTP server
- `bin/service` : the service manager (start/stop/reload, daemon mode)
- `bin/userctl` : CLI for user management
- `bin/datasource-sync` : Connect to your Apache Druid to map datasource

---

## Configuration

- `config.yaml`: server parameters, JWT, user backend, logs, static files
- `druid.yaml`: all datasources, dimensions and metrics mapping (with custom formulas)
- `users.yaml`: users (if backend is "file"), with hash/salt/admin

See [docs/configuration.md](docs/configuration.md) for details.

---

## Documentation

- [Configuration](docs/configuration.md)
- [REST API](docs/api.md)
- [Static file customization](docs/static.md)
- [Security & access control](docs/security.md)

---

## Security & Rights

- JWT Bearer authentication (JWT stored in client local storage)
- Any `reserved: true` dimension or metric in `druid.yaml` is admin-only
- All rights are checked at each report execution request (401/403 + detailed log)
- Static files are strictly whitelisted (wildcard supported), fallback to admin/static_default

---

## Custom metric formulas

- Supports complex arithmetic formulas for metrics (`cpm: 1000 * revenue / impressions`, etc.), translated **directly into Druid postAggregations**.
- Parsing is secured: it is impossible to use undeclared or reserved metrics/dimensions without the proper rights.

---

## Testing

Run all unit tests:

```sh
make test
# or
go test ./...
```

---

## Extending

- To add metrics/dimensions, simply update `druid.yaml` (supports formulas and mapping).
- To change rights, edit the `reserved` flag or set users as `admin: true/false`.
- To use a SQL backend for users, set `auth.user_backend` and a SQL query in `config.yaml`.

---

## Contributing

Contributions are welcome!  
Please read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting a pull request.

---

## License

MIT

---

**Need help with configuration, an extension or the code?  
Contact the maintainer or open an issue.**

---
