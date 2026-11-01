---
sidebar_position: 3
---

# Configuration

BookLife uses [KDL](https://kdl.dev) format for configuration. The config file is specified with `--config` flag.

## Full Reference

```kdl
// BookLife MCP Server Configuration

server {
    name "booklife"
    version "0.1.0"
    transport "stdio"
}

user {
    name "Reader"
    timezone "America/Chicago"

    preferences {
        genres "literary-fiction" "sci-fi" "fantasy" "non-fiction"
        avoid-genres "romance"
        preferred-formats "ebook" "audiobook" "physical"
        max-tbr-size 50
    }
}

providers {
    hardcover enabled=true {
        // Use env= to reference environment variables (recommended)
        api-key env="HARDCOVER_API_KEY"
        endpoint "https://api.hardcover.app/v1/graphql"

        sync {
            auto-import-libby true
            default-status "want-to-read"
        }
    }

    libby enabled=true {
        // Libraries are configured automatically via 'booklife libby-connect'
        notifications {
            hold-available true
            due-soon-days 3
        }
    }

    open-library enabled=true {
        endpoint "https://openlibrary.org"
        covers-endpoint "https://covers.openlibrary.org"
        rate-limit-ms 100
    }

    // Future providers (not yet implemented)
    wikidata enabled=false
    youtube enabled=false
    tiktok enabled=false
    local-bookstores enabled=false
}

cache {
    path "~/.booklife/cache.db"

    book-metadata true
    cover-images true
    reading-history true

    embeddings {
        enabled false
        model "all-MiniLM-L6-v2"
        index-path "~/.booklife/embeddings.idx"
    }
}

features {
    semantic-search false
    auto-hold false
    mood-tracking false
}
```

## Environment Variables

Configuration values can reference environment variables using the `env="VAR_NAME"` syntax:

```kdl
hardcover enabled=true {
    api-key env="HARDCOVER_API_KEY"
}
```

### Available Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `HARDCOVER_API_KEY` | Yes | Hardcover GraphQL API token |
| `BOOKLIFE_DATA_DIR` | No | Override data directory (default: `~/.local/share/booklife`) |
| `BOOKLIFE_CONFIG_PATH` | No | Config file path (default: `./booklife.kdl`) |
| `BOOKLIFE_LOG_LEVEL` | No | Log level: `debug`, `info`, `warn`, `error` |

## Data Directory

BookLife stores local data (history, TBR, enrichment) in SQLite databases:

- **Linux**: `~/.local/share/booklife/`
- **macOS**: `~/Library/Application Support/booklife/`

Override with `BOOKLIFE_DATA_DIR` environment variable.

## Libby TLS Note

Two OverDrive endpoints serve certificates issued for a different hostname. BookLife verifies them against the correct name, so no configuration is needed. The old `skip-tls-verify` option is ignored, and certificates are always verified.

## Provider Initialization

Providers are conditionally initialized based on their `enabled` flag:

- If a provider is disabled, its tools won't be registered
- If Libby has no saved identity, a warning is printed but the server still starts
- All providers can fail independently without crashing the server
