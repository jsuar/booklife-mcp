# BookLife MCP

**Your reading life, unified.** One MCP server that connects your library, reading tracker, and bookshelf into a seamless AI-powered reading assistant.

BookLife bridges [Hardcover](https://hardcover.app) (reading tracker), [Libby/OverDrive](https://libbyapp.com) (library access), and [Open Library](https://openlibrary.org) (metadata) so you can discover books, borrow from the library, track your reading, and get personalized recommendations — all through natural conversation with Claude.

---

## Quick Start

### 1. Build

```bash
cd booklife-mcp
go build -o booklife ./cmd/booklife
```

### 2. Connect Libby

```bash
# Get your clone code from Libby app:
# Settings → Copy To Another Device → Sonos Speakers
./booklife libby-connect <8-digit-code>
```

### 3. Configure

Create `booklife.kdl` (copy `booklife.kdl.example`; the real file is gitignored, so keep keys in environment variables):

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

### 4. Add to Claude Desktop

```json
{
  "mcpServers": {
    "booklife": {
      "command": "/path/to/booklife",
      "args": ["--config", "/path/to/booklife.kdl"],
      "env": {
        "HARDCOVER_API_KEY": "your-key-from-hardcover.app/settings/api"
      }
    }
  }
}
```

### 5. Install the Claude Code Plugin (Optional)

For enhanced Claude Code integration with skills and slash commands:

```bash
# Add the marketplace
/plugin marketplace add https://github.com/andylbrummer/andy-marketplace

# Install the plugin
/plugin install booklife@andy-marketplace
```

See the [BookLife plugin in andy-marketplace](https://github.com/andylbrummer/andy-marketplace/tree/main/plugins/booklife) for details.

---

## Features

### Library Access (Libby/OverDrive)
Search your library catalog, check availability, place holds on ebooks and audiobooks, track loans and due dates — all without opening the Libby app.

### Reading Tracker (Hardcover)
Manage your reading list, update progress, rate books, and maintain your reading history on Hardcover through natural language.

### Unified TBR Management
A single to-be-read list that aggregates books from Hardcover (want-to-read), Libby (holds and tags), and manually added physical books. Filter by source, search, and prioritize.

### Comprehensive Sync
One-command sync that chains: import Libby loans → mark returned books as "read" in Hardcover → enrich metadata → cache Libby tags. Incremental, with dry-run preview.

### Content-Based Recommendations
Enriches your reading history with themes, topics, and mood data from Open Library and Google Books, then finds similar books based on what you've enjoyed.

### Reading Profile & Analytics
Automatic analysis of your reading patterns: format preferences, top genres, favorite authors, reading cadence, streaks, and completion rates.

### Progressive Discovery
Built-in `info` tool with workflow guides, category browsing, and detailed tool help. Never wonder what's available — just ask.

---

## Tools at a Glance

| Category | Tools | Purpose |
|----------|-------|---------|
| **Hardcover** | 3 | Library management, status updates, book tracking |
| **Libby** | 6 | Catalog search, loans, holds, tag sync |
| **TBR** | 6 | Unified reading list across all sources |
| **Unified** | 2 | Cross-provider search and access recommendations |
| **History** | 4 | Timeline import, local store, statistics |
| **Enrichment** | 2 | Background metadata jobs with progress tracking |
| **Sync** | 1 | Universal sync with progressive disclosure |
| **Profile** | 1 | Reading preferences and patterns |
| **Recommendations** | 1 | Content-based book similarity |
| **Info** | 1 | Progressive discovery and workflow guides |

---

## Architecture

```
┌──────────────────────────────────────────┐
│           Claude / AI Assistant          │
└────────────────────┬─────────────────────┘
                     │ MCP (stdio)
┌────────────────────▼─────────────────────┐
│           BookLife MCP Server            │
│                                          │
│  ┌─────────┐ ┌─────────┐ ┌───────────┐  │
│  │Hardcover│ │  Libby  │ │OpenLibrary│  │
│  │ GraphQL │ │   API   │ │  REST API │  │
│  └─────────┘ └─────────┘ └───────────┘  │
│                                          │
│  ┌──────────────────────────────────┐    │
│  │        Local SQLite Store        │    │
│  │  History · TBR · Enrichment      │    │
│  └──────────────────────────────────┘    │
└──────────────────────────────────────────┘
```

---

## Documentation

Full documentation is available at **[andylbrummer.github.io/booklife-mcp](https://andylbrummer.github.io/booklife-mcp)**:

- [Getting Started](https://andylbrummer.github.io/booklife-mcp/docs/getting-started) — Installation and configuration
- [Tool Reference](https://andylbrummer.github.io/booklife-mcp/docs/category/tool-reference) — All 27 tools with parameters and examples
- [Workflows](https://andylbrummer.github.io/booklife-mcp/docs/category/workflows) — Step-by-step guides for common tasks
- [Configuration](https://andylbrummer.github.io/booklife-mcp/docs/configuration) — KDL config file reference
- [Claude Code Plugin](https://andylbrummer.github.io/booklife-mcp/docs/claude-code-plugin) — Skills and commands via andy-marketplace

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `HARDCOVER_API_KEY` | Yes | [Get from Hardcover settings](https://hardcover.app/settings/api) |
| `BOOKLIFE_DATA_DIR` | No | Data directory (default: `~/.local/share/booklife`) |
| `BOOKLIFE_LOG_LEVEL` | No | `debug` / `info` / `warn` / `error` |

Libby authentication is handled via the `booklife libby-connect` command (no environment variable needed).

---

## CLI Commands

```bash
booklife serve --config booklife.kdl    # Start MCP server
booklife libby-connect <code>           # Connect Libby account
booklife sync [--dry-run] [--limit N]   # Sync returned books to Hardcover
booklife import-timeline <file>         # Import Libby timeline JSON
booklife version                        # Show version
```

---

## Development

```bash
# Build
cd booklife-mcp && go build -o booklife ./cmd/booklife

# Run tests
go test ./...

# Vet
go vet ./...
```

---

## License

Personal project — adapt as needed for your own reading management.
