# Static File Customization

## Whitelist and Security

- Only files listed in `static_allowed` (wildcards supported) in `config.yaml` are served.
- Example: `"js/*.js"` allows all JS files in the `js/` directory.

## Fallback

- If a file is not found in the main static directory (`static`), it is searched in the fallback directory (`static_default`).
- This allows easy theming or overriding of UI files.

## Template Macros

- Static files can contain variables like `{APP_NAME}`.
- These are replaced at runtime using values from the configuration (see `templateVars` if present).

## Example

- `static/index.html`: Custom version (if present).
- `static_default/index.html`: Default version (used as fallback).

## Logging

- All static file accesses (OK, refused, not found) are logged in `access.log`.

---