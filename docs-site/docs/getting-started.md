---
sidebar_position: 2
---

# Getting Started

## Prerequisites

- **Go 1.21+** — For building the server
- **Hardcover account** — [Sign up](https://hardcover.app) and get an API key from [Settings > API](https://hardcover.app/settings/api)
- **Libby account** — Library card connected to the [Libby app](https://libbyapp.com)

## Installation

### Build from Source

```bash
git clone https://github.com/andylbrummer/booklife-mcp.git
cd booklife-mcp/booklife-mcp
go build -o booklife ./cmd/booklife
```

### Connect Libby

Libby authentication uses a clone code from the app:

1. Open Libby app on your phone
2. Go to **Settings** → **Copy To Another Device** → **Sonos Speakers**
3. Note the 8-digit code (you have ~40 seconds)
4. Run:

```bash
./booklife libby-connect <8-digit-code>
```

The identity is saved to `~/.config/booklife/libby-identity.json`.

### Set Environment Variables

```bash
export HARDCOVER_API_KEY="your-key-from-hardcover-settings"
```

## Configuration

Create a `booklife.kdl` configuration file from `booklife.kdl.example`:

```kdl
server {
    name "booklife"
    version "0.1.0"
    transport "stdio"
}

providers {
    hardcover enabled=true {
        api-key env="HARDCOVER_API_KEY"
        endpoint "https://api.hardcover.app/v1/graphql"
    }

    libby enabled=true {
        notifications {
            hold-available true
            due-soon-days 3
        }
    }

    open-library enabled=true {
        endpoint "https://openlibrary.org"
        covers-endpoint "https://covers.openlibrary.org"
    }
}
```

See [Configuration](./configuration) for the full reference.

## Add to Claude Desktop

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "booklife": {
      "command": "/path/to/booklife",
      "args": ["--config", "/path/to/booklife.kdl"],
      "env": {
        "HARDCOVER_API_KEY": "your-api-key"
      }
    }
  }
}
```

## Verify

Start a conversation with Claude and ask:

> "Use the info tool to show me what BookLife can do."

You should see the full category overview with available tools and workflows.

## Import Reading History (Optional)

If you have Libby reading history to import:

1. In Libby app: **Settings** → **Reading History** → **Export Timeline**
2. Copy the JSON URL
3. Use the `history_import_timeline` tool or CLI:

```bash
./booklife import-timeline /path/to/timeline.json --config booklife.kdl
```

## Next Steps

- Run `sync` with `action="sync_all"` to do a comprehensive initial sync
- Use `profile_get` to see your reading profile
- Try `booklife_find_book_everywhere` to search across all sources
