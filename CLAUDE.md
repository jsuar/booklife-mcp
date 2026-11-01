# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

BookLife is an MCP (Model Context Protocol) server for managing a personal reading life across multiple platforms. It unifies Hardcover (reading tracker), Libby/OverDrive (library access), Open Library (metadata), and local bookstores into a cohesive assistant that prioritizes free/local options.

## Build and Run Commands

```bash
# Build the server
cd booklife-mcp && go build -o booklife ./cmd/booklife

# Run the server (copy booklife.kdl.example to booklife.kdl first; never commit it)
./booklife serve --config booklife.kdl

# Run tests
go test ./...

# Run a single test
go test -run TestName ./internal/...
```

## Environment Variables

Required:
- `HARDCOVER_API_KEY` - Hardcover GraphQL API token

Libby does not use an environment variable: run `booklife libby-connect <8-digit-code>` once to save an identity.

Optional:
- `YOUTUBE_API_KEY` - For BookTube integration
- `BOOKLIFE_CONFIG_PATH` - Config file path (default: `./booklife.kdl`)
- `BOOKLIFE_LOG_LEVEL` - debug/info/warn/error

## Architecture

```
booklife-mcp/
├── cmd/booklife/main.go     # Entry point, signal handling
├── internal/
│   ├── config/              # KDL configuration parsing with env var resolution
│   ├── server/              # MCP server setup
│   │   ├── server.go        # Server struct, provider initialization
│   │   ├── tools.go         # MCP tool handlers (search_books, get_my_library, etc.)
│   │   ├── resources.go     # MCP resource handlers (booklife://library/*, etc.)
│   │   └── prompts.go       # MCP prompt templates (what_should_i_read, etc.)
│   ├── providers/           # External API clients
│   │   ├── hardcover/       # GraphQL client for Hardcover
│   │   ├── libby/           # Reverse-engineered Libby/OverDrive API
│   │   └── openlibrary/     # Open Library REST API with rate limiting
│   └── models/              # Shared types (Book, LibbyLoan, LibraryAvailability, etc.)
```

### Key Patterns

**Provider Initialization**: Providers are conditionally initialized in `server.initProviders()` based on config `Enabled` flags. Tools check for nil providers before executing.

**MCP SDK Usage**: Uses `github.com/modelcontextprotocol/go-sdk` with typed tool handlers:
```go
mcp.AddTool(s.mcpServer, &mcp.Tool{Name: "tool_name", Description: "..."}, s.handler)
```

**Configuration**: KDL v1 format with environment variable resolution (`api-key env="VAR_NAME"` syntax). See SPEC.md for full config structure.

**Rate Limiting**: Open Library client uses `golang.org/x/time/rate` for 10 req/sec limiting.

**Enrichment Strategy**: Hardcover → Open Library → Google Books fallback chain for book metadata enrichment. Hardcover provides ISBN lookup and rich metadata (genres, tags, series), while OL/GB provide subject/category data when Hardcover lacks the book.

## Hardcover GraphQL API

**Endpoint**: `https://api.hardcover.app/v1/graphql`

**Authentication**: Bearer token in `Authorization` header
```
Authorization: Bearer YOUR_API_KEY
```

**Get API Key**: https://hardcover.app/settings/api

### Available Enrichment Fields

Based on Hardcover's GraphQL schema (current as of December 2025), books support these enrichment fields:

```graphql
{
  books(where: {id: {_eq: $id}}) {
    id
    title
    subtitle
    description
    pages
    release_date
    rating
    ratings_count
    cached_image  # JSON object with { url: string } structure
    cached_tags   # JSON field containing all tags organized by category

    # Authors
    contributions {
      author {
        id
        name
      }
    }

    # ISBNs
    editions {
      isbn_10
      isbn_13
    }

    # Series information
    book_series {
      series {
        name
      }
      position
    }
  }
}
```

### Tag System (cached_tags)

**IMPORTANT:** As of December 2025, Hardcover uses `cached_tags` (a JSON field) instead of the old `tags` relationship.

The `cached_tags` field returns a JSON object with these categories:
- **Genre**: Fiction, Mystery, Science Fiction, Fantasy, etc.
- **Mood**: Dark, Hopeful, Funny, Mysterious, etc.
- **ContentWarning**: Content warnings flagged by users

**Note:** The old "theme" category no longer exists in the API. Themes should be derived from genres/moods or extracted from descriptions.

Each tag entry includes:
```json
{
  "tag": "Science Fiction",
  "tagSlug": "science-fiction",
  "category": "Genre",
  "categorySlug": "genre",
  "spoiler": false,
  "count": 150
}
```

Access tags in code:
```go
// cached_tags is a map[string][]Tag
genres := book.CachedTags["Genre"]
moods := book.CachedTags["Mood"]
```

### Enrichment Query Pattern

For efficient enrichment, use this query to fetch all metadata:

```graphql
query GetBookEnrichment($id: Int!) {
  books(where: {id: {_eq: $id}}, limit: 1) {
    id
    title
    subtitle
    description
    pages
    release_date
    cached_image
    cached_tags
    contributions {
      author { name }
    }
    editions {
      isbn_10
      isbn_13
    }
    book_series {
      series { name }
      position
    }
  }
}
```

### Rate Limits

Hardcover doesn't publicly document rate limits, but reasonable usage:
- Use batched queries when possible
- Cache results to minimize API calls
- Implement exponential backoff on errors

### Integration Tests

Run enrichment field discovery tests with:
```bash
HARDCOVER_API_KEY=your_key go test -v -run TestBookEnrichmentFields ./internal/providers/hardcover/
```

## MCP Interface

### Tools
- **Library Management**: `search_books`, `get_my_library`, `update_reading_status`, `add_to_library`
- **Libby**: `search_library`, `check_availability`, `get_loans`, `get_holds`, `place_hold`
- **Unified**: `find_book_everywhere`, `best_way_to_read`, `add_to_tbr`

### Resources
- `booklife://library/current` - Currently reading books
- `booklife://library/tbr` - Want-to-read list
- `booklife://loans` - Active Libby loans
- `booklife://holds` - Library hold queue
- `booklife://stats` - Reading statistics

### Prompts
- `what_should_i_read` - Personalized recommendations
- `book_summary` - Book summaries with spoiler control
- `reading_wrap_up` - Period reading summaries
- `pick_from_tbr` - TBR decision helper

## Libby/OverDrive TLS Certificates

Two OverDrive endpoints serve a certificate that does not cover their public hostname, so `tlsServerNames` in `internal/providers/libby/client.go` sets the name to verify instead. `thunder.api.overdrive.com` needs no override.

| Endpoint | Verified as |
|---|---|
| `sentry-read.svc.overdrive.com` | `sentry-read.odrsre.overdrive.com` |
| `vandal.svc.overdrive.com` | `vandal.libbyapp.com` |
| `thunder.api.overdrive.com` | its own hostname |

Certificates are always verified. **Never disable verification**: these connections carry the clone code and the Libby session token. If a certificate name changes, update the map. To check the live hosts:

```bash
BOOKLIFE_LIVE_TESTS=1 go test -run TestLiveTLS ./internal/providers/libby/
```

## Implementation Status

See SPEC.md for full implementation phases. Current state:
- Phase 1-2: Core infrastructure and Hardcover integration (implemented)
- Phase 3: Libby integration (implemented)
- Phase 4: Unified actions (partially implemented)
- Phase 5-6: Community/semantic features (not started)

## Claude Desktop Integration

```json
{
  "mcpServers": {
    "booklife": {
      "command": "/path/to/booklife",
      "args": ["--config", "/path/to/booklife.kdl"],
      "env": {
        "HARDCOVER_API_KEY": "..."
      }
    }
  }
}
```
