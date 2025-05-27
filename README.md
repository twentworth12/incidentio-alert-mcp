# incident.io Alert MCP

A Model Context Protocol (MCP) server for sending alerts to incident.io.

## Features

- Send alerts to incident.io via MCP tools
- Support for alert deduplication
- Configurable webhook URL and API token
- Metadata support for additional context

## Installation

1. Clone the repository:
```bash
git clone https://github.com/twentworth12/incidentio-alert-mcp.git
cd incidentio-alert-mcp
```

2. Build the server:
```bash
make build
```

## Configuration

Set the following environment variables:

- `INCIDENTIO_WEBHOOK_URL`: Your incident.io webhook URL
- `INCIDENTIO_API_TOKEN`: Your incident.io API token

Copy `.env.example` to `.env` and update with your values:
```bash
cp .env.example .env
```

## Usage

### Running the server

```bash
make run
```

### MCP Tool: send_alert

Send an alert to incident.io with the following parameters:

- `title` (required): Alert title
- `deduplication_key` (required): Unique key to deduplicate alerts
- `description` (optional): Alert description
- `status` (optional): Alert status ("firing" or "resolved"), defaults to "firing"
- `metadata` (optional): Additional metadata as key-value pairs

Example:
```json
{
  "title": "Database Connection Error",
  "deduplication_key": "db-conn-error-prod",
  "description": "Unable to connect to primary database",
  "status": "firing",
  "metadata": {
    "team": "backend",
    "service": "api",
    "environment": "production"
  }
}
```

## Development

- `make build` - Build the binary
- `make run` - Run the server
- `make test` - Run tests
- `make fmt` - Format code
- `make lint` - Lint code (requires golangci-lint)

## License

MIT