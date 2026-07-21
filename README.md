# gsc-cli

A lightweight, premium command-line tool for [Google Search Console](https://search.google.com/search-console/about), written in Go. Manage verified properties, submit sitemaps, query search performance analytics, and inspect URL indexing status directly from your terminal.

## Features

- **Sites** — list, add, and delete verified Search Console properties
- **Sitemaps** — list, submit, and delete sitemaps for a property
- **Query** — query clicks, impressions, CTR, and average position; group by query, page, country, device, date, or search appearance; filter results; export to table, CSV, or JSON
- **Inspect** — check a URL's live index status, canonical URL, crawl info, and mobile usability, one URL at a time or in batch from a file

## Installation

Requires [Go](https://go.dev/dl/) 1.26 or later.

Install directly with Go:

```bash
go install github.com/bertramdev/gsc-cli@latest
```

This places a `gsc-cli` binary in `$(go env GOPATH)/bin`. Make sure that directory is on your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

Alternatively, build from a clone of the repository:

```bash
git clone https://github.com/bertramdev/gsc-cli.git
cd gsc-cli
go build -o gsc-cli .
```

Move the resulting `gsc-cli` binary onto your `PATH` (e.g. `/usr/local/bin`) to use it from anywhere.

## Authentication

`gsc-cli` talks to the Search Console (Webmasters + URL Inspection) APIs and needs Google credentials. It supports three methods, tried in this order:

1. **Service account key** — pass a JSON key file via `-c/--credentials`, or set `GOOGLE_APPLICATION_CREDENTIALS`.
   The service account must be added as a user on the target property in Search Console.
2. **OAuth2 client secret** — pass a Google OAuth2 client secret JSON file (Desktop app type) via `-s/--client-secret`. On first use, `gsc-cli` opens a browser for you to authorize, then caches the resulting token at `~/.gsc-cli/token.json`. The client secret itself is cached at `~/.gsc-cli/client_secret.json` so you only need to pass `-s` once.
3. **Application Default Credentials** — if neither of the above is provided, `gsc-cli` falls back to [ADC](https://cloud.google.com/docs/authentication/application-default-credentials) (e.g. credentials from `gcloud auth application-default login`).

To create an OAuth2 client secret: in [Google Cloud Console](https://console.cloud.google.com/apis/credentials), create an OAuth client ID of type **Desktop app**, enable the **Search Console API**, and download the JSON file.

**Never commit credential files** (service account keys, client secrets, or cached tokens) to version control. `*credentials*.json`, `*secret*.json`, and `token.json` are already excluded via `.gitignore`.

## Usage

Every command that operates on a specific property requires `-u/--site` with the property's verified URL as it appears in Search Console (e.g. `sc-domain:example.com` or `https://example.com/`).

### Safety & Non-Interactive Environments (AI Agent Compatibility)

Destructive commands (`sites delete` and `sitemaps delete`) protect against accidental data loss by prompting for confirmation in interactive terminals.

When run in a **non-interactive/headless environment** (such as CI/CD pipelines, cron jobs, or by autonomous **AI agents**), the command will safely fail unless the `--force` (shorthand `-y`) flag is supplied:

```bash
# Safe bypass:
gsc-cli sites delete sc-domain:example.com --force
```

### Sites

```bash
gsc-cli sites list
gsc-cli sites add sc-domain:example.com

# Prompts for confirmation in interactive mode
gsc-cli sites delete sc-domain:example.com

# Force delete without confirmation (ideal for AI agents and scripts)
gsc-cli sites delete sc-domain:example.com --force
```

### Sitemaps

```bash
gsc-cli sitemaps list -u sc-domain:example.com
gsc-cli sitemaps add https://example.com/sitemap.xml -u sc-domain:example.com

# Prompts for confirmation in interactive mode
gsc-cli sitemaps delete https://example.com/sitemap.xml -u sc-domain:example.com

# Force delete without confirmation (ideal for AI agents and scripts)
gsc-cli sitemaps delete https://example.com/sitemap.xml -u sc-domain:example.com --force
```

### Query

```bash
# Top queries for the last 30 days
gsc-cli query -u sc-domain:example.com

# Group by page and country, filter to a specific country, export as CSV
gsc-cli query -u sc-domain:example.com \
  --dimensions page,country \
  --filter "country equals usa" \
  --start-date 2026-06-01 --end-date 2026-06-30 \
  --limit 500 \
  --format csv --output-file results.csv
```

Flags:

| Flag | Description |
|---|---|
| `-d, --start-date` | Start date `YYYY-MM-DD` (default: 30 days before end date) |
| `-e, --end-date` | End date `YYYY-MM-DD` (default: 2 days ago) |
| `--dimensions` | Comma-separated: `query`, `page`, `country`, `device`, `date`, `searchAppearance` |
| `--filter` | `"<dimension> <operator> <value>"`, repeatable. Operators: `contains`, `equals`/`==`, `notContains`, `notEquals`/`!=`, `includingRegex`/`~`, `excludingRegex`/`!~` |
| `--limit` | Max rows to retrieve (default: 100) |
| `-f, --format` | `table`, `csv`, or `json` (default: `table`) |
| `-o, --output-file` | Write results to a file instead of stdout |

### Inspect

```bash
# Single URL
gsc-cli inspect https://example.com/page -u sc-domain:example.com

# Batch from a file (one URL per line, '#' or '//' lines ignored)
gsc-cli inspect --file urls.txt -u sc-domain:example.com --format json
```

## Global flags

| Flag | Description |
|---|---|
| `-c, --credentials` | Path to a service account JSON key file |
| `-s, --client-secret` | Path to an OAuth2 client secret JSON file |
| `-u, --site` | Verified property URL |
| `-v, --verbose` | Enable verbose logging |

## Development

```bash
go test ./...
```

## License

Licensed under the [Apache License 2.0](LICENSE).
