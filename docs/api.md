# REST API

## Authentication

- `POST /api/login`  
  Authenticate with username and password. Returns a JWT.

**Request payload:**
```json
{
  "username": "alice",
  "password": "your_password"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

---

## Schema

- `GET /api/schema`  
  Returns the available datasources, dimensions, and metrics for the current user.

**Response:**
```json
{
  "myreport": {
    "dimensions": ["date", "browser", "device"],
    "metrics": [
      {"name": "errors", "type": "long"},
      {"name": "requests", "type": "long"},
      {"name": "errorrate", "type": "float"}
    ]
  }
}
```

---

## Reports

- `POST /api/reports/execute`  
  Launch an asynchronous report (JSON payload: datasource, dimensions, metrics, filters). Returns a report ID.

**Request payload:**
```json
{
  "datasource": "myreport",
  "dimensions": ["date", "browser"],
  "metrics": ["requests", "errors"],
  "filters": {
    "browser": ["Chrome", "Firefox"],
    "date": {"from": "2024-01-01", "to": "2024-01-31"}
  }
}
```

**Response:**
```json
{
  "id": "report_1234567890"
}
```

---

- `GET /api/reports/status?id=...`  
  Get the status of a report (waiting/processing/complete/error) and retrieve the result if ready.

**Response (example):**
```json
{
  "status": "complete"
}
```

---

- `GET /api/reports/download?id=...&type=csv|excel`  
  Download the result of a completed report.

**Parameters:**
- `id` (required): report ID
- `type` (optional): `csv` (default) or `excel` (for XLSX format)

**Response:**  
Returns a CSV or Excel file as attachment.

---

## Filters

- `POST /api/filters/values`  
  Get possible values for a dimension (for filter auto-completion).

**Request payload:**
```json
{
  "datasource": "myreport",
  "dimension": "browser",
  "date_start": "2024-01-01",   // optional, filter values in this interval
  "date_end": "2024-01-31"      // optional, filter values in this interval
}
```

**Response:**
```json
{
  "values": ["Chrome", "Firefox", "Safari"]
}
```

---

## Static files

- Served via `/` (root path), only if whitelisted in `static_allowed`.

---

## Security

- All API endpoints (except `/api/login`) require a valid JWT in the `Authorization: Bearer ...` header.

---
