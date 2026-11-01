# BookLife MCP Server Specification

> A unified MCP server for managing your reading life across Hardcover, Libby/OverDrive, local bookstores, and book community platforms.

## Vision

Replace the fragmented book ecosystem (Goodreads, Amazon, scattered library apps) with a cohesive personal reading assistant that:

1. **Tracks** your reading across all platforms (Hardcover as source of truth)
2. **Discovers** books through library availability, local stores, BookTok/BookTube trends
3. **Connects** semantic book relationships beyond simple genre tags
4. **Prioritizes** free/local options (library first, then indie bookstores, then online)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Claude / MCP Client                       │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                     BookLife MCP Server (Go)                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Tools     │  │  Resources  │  │   Prompts   │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│         │                │                │                      │
│         ▼                ▼                ▼                      │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                    Provider Adapters                        ││
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       ││
│  │  │Hardcover │ │  Libby   │ │OpenLibrary│ │ YouTube  │       ││
│  │  │ GraphQL  │ │  (APL)   │ │   API    │ │Data API  │       ││
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘       ││
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       ││
│  │  │ Wikidata │ │BookPeople│ │  TikTok  │ │  Local   │       ││
│  │  │ SPARQL   │ │ Scraper  │ │ Scraper  │ │  Cache   │       ││
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘       ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

---

## Configuration (KDL)

Configuration uses KDL v1 format via `github.com/sblinch/kdl-go`.

### `booklife.kdl`

```kdl
// BookLife MCP Server Configuration

server {
    name "booklife"
    version "0.1.0"
    transport "stdio"  // or "sse" for HTTP
}

// User identity and preferences
user {
    name "Your Name"
    timezone "America/Chicago"
    
    // Reading preferences for recommendations
    preferences {
        genres "literary-fiction" "sci-fi" "non-fiction"
        avoid-genres "romance" "horror"
        preferred-formats "ebook" "audiobook" "physical"
        max-tbr-size 50  // Cap TBR list recommendations
    }
}

// Provider configurations
providers {
    
    // Hardcover - Primary reading tracker
    hardcover enabled=true {
        api-key env="HARDCOVER_API_KEY"
        endpoint "https://api.hardcover.app/v1/graphql"
        
        // Sync settings
        sync {
            auto-import-libby true
            default-status "want-to-read"  // when adding from other sources
        }
    }
    
    // Libby/OverDrive - Library access
    libby enabled=true {
        // Libraries are synced automatically via 'booklife libby-connect'

        // Notification preferences
        notifications {
            hold-available true
            due-soon-days 3
        }
    }
    
    // Open Library - Free metadata
    open-library enabled=true {
        endpoint "https://openlibrary.org"
        covers-endpoint "https://covers.openlibrary.org"
        rate-limit-ms 100
    }
    
    // Wikidata - Knowledge graph queries
    wikidata enabled=true {
        sparql-endpoint "https://query.wikidata.org/sparql"
        // Used for author relationships, literary movements, etc.
    }
    
    // YouTube Data API - BookTube content
    youtube enabled=true {
        api-key env="YOUTUBE_API_KEY"
        
        // Channels to monitor for recommendations
        booktube-channels {
            channel "UCvO6R5CzC7pbSZPmEPPhVOQ"  // Example channel ID
            channel "UC2Z7QaxoG3P2vJtdGnJuzCg"
        }
    }
    
    // TikTok/BookTok - Scraping (optional, requires paid service)
    tiktok enabled=false {
        scraper-api env="TIKTOK_SCRAPER_API_KEY"
        hashtags "#BookTok" "#BookRecommendations" "#DarkRomance"
    }
    
    // Local bookstores (web scraping)
    local-bookstores enabled=true {
        
        store "bookpeople" {
            name "BookPeople"
            website "https://bookpeople.com"
            location "603 N. Lamar, Austin, TX"
            phone "512-472-5050"
            events-url "https://bookpeople.com/events"
            
            // Search inventory via their website
            search-enabled true
        }
        
        store "bookwoman" {
            name "BookWoman"
            website "https://bookwoman.com"
            location "5501 N Lamar Blvd, Austin, TX"
            specialties "feminist" "lgbtq" "local-authors"
        }
        
        store "black-pearl" {
            name "Black Pearl Books"
            website "https://blackpearlbookstore.com"
            location "Austin, TX"
            specialties "bipoc" "diverse-voices"
        }
    }
}

// Local cache/database settings
cache {
    path "~/.booklife/cache.db"  // SQLite
    
    // What to cache locally
    book-metadata true
    cover-images true
    reading-history true
    
    // Embedding index for semantic search
    embeddings {
        enabled true
        model "all-MiniLM-L6-v2"  // sentence-transformers
        index-path "~/.booklife/embeddings.idx"
    }
}

// Feature flags
features {
    // Experimental features
    semantic-search false  // Enable embedding-based book similarity
    auto-hold false        // Automatically place holds when TBR books available
    mood-tracking false    // Track reading moods over time
}
```

---

## MCP Tools

### Library Management (Hardcover)

```go
// Tool: search_books
// Search for books across all metadata sources
type SearchBooksInput struct {
    Query    string   `json:"query" jsonschema:"required,description=Title, author, or ISBN"`
    Sources  []string `json:"sources,omitempty" jsonschema:"description=hardcover|openlibrary|wikidata"`
    Limit    int      `json:"limit,omitempty" jsonschema:"default=10,maximum=50"`
}

// Tool: get_my_library
// Get user's reading list from Hardcover
type GetMyLibraryInput struct {
    Status   string `json:"status,omitempty" jsonschema:"enum=reading|read|want-to-read|dnf|all"`
    SortBy   string `json:"sort_by,omitempty" jsonschema:"enum=date_added|title|author|rating"`
    Limit    int    `json:"limit,omitempty" jsonschema:"default=20"`
}

// Tool: update_reading_status  
// Update a book's status in Hardcover
type UpdateReadingStatusInput struct {
    BookID   string `json:"book_id" jsonschema:"required,description=Hardcover book ID"`
    Status   string `json:"status" jsonschema:"required,enum=reading|read|want-to-read|dnf"`
    Progress int    `json:"progress,omitempty" jsonschema:"description=Percentage 0-100"`
    Rating   float64 `json:"rating,omitempty" jsonschema:"description=Rating 0-5 with 0.25 increments"`
}

// Tool: add_to_library
// Add a book to Hardcover library
type AddToLibraryInput struct {
    ISBN     string `json:"isbn,omitempty" jsonschema:"description=ISBN-10 or ISBN-13"`
    Title    string `json:"title,omitempty" jsonschema:"description=Book title if ISBN unknown"`
    Author   string `json:"author,omitempty" jsonschema:"description=Author name if ISBN unknown"`
    Status   string `json:"status,omitempty" jsonschema:"default=want-to-read"`
    PlaceHold bool  `json:"place_hold,omitempty" jsonschema:"description=Also place library hold if available"`
}

// Tool: get_recommendations
// Get personalized book recommendations
type GetRecommendationsInput struct {
    BasedOn    string   `json:"based_on,omitempty" jsonschema:"description=Book ID or recent-reads or similar-users"`
    Genres     []string `json:"genres,omitempty"`
    AvailableAt string  `json:"available_at,omitempty" jsonschema:"enum=library|any"`
    Limit      int      `json:"limit,omitempty" jsonschema:"default=10"`
}
```

### Library Access (Libby/APL)

```go
// Tool: search_library
// Search Austin Public Library catalog via Libby
type SearchLibraryInput struct {
    Query      string   `json:"query" jsonschema:"required"`
    Format     []string `json:"format,omitempty" jsonschema:"description=ebook|audiobook|all"`
    Available  bool     `json:"available,omitempty" jsonschema:"description=Only show currently available"`
}

// Tool: check_availability
// Check if a specific book is available at the library
type CheckAvailabilityInput struct {
    ISBN     string `json:"isbn,omitempty"`
    Title    string `json:"title,omitempty"`  
    Author   string `json:"author,omitempty"`
}

// Tool: get_loans
// Get current Libby loans
type GetLoansInput struct {
    IncludeExpired bool `json:"include_expired,omitempty"`
}

// Tool: get_holds  
// Get current hold queue
type GetHoldsInput struct{}

// Tool: place_hold
// Place a hold on a library book
type PlaceHoldInput struct {
    MediaID  string `json:"media_id" jsonschema:"required,description=OverDrive media ID"`
    Format   string `json:"format" jsonschema:"required,enum=ebook|audiobook"`
    AutoBorrow bool `json:"auto_borrow,omitempty" jsonschema:"default=true"`
}

// Tool: return_loan
// Return a library book early
type ReturnLoanInput struct {
    LoanID string `json:"loan_id" jsonschema:"required"`
}

// Tool: get_reading_progress
// Get reading progress from Libby
type GetReadingProgressInput struct {
    LoanID string `json:"loan_id" jsonschema:"required"`
}
```

### Local Bookstores

```go
// Tool: search_local_bookstore
// Search inventory at local Austin bookstores
type SearchLocalBookstoreInput struct {
    Query    string   `json:"query" jsonschema:"required"`
    Store    string   `json:"store,omitempty" jsonschema:"description=bookpeople|bookwoman|black-pearl|all"`
}

// Tool: get_bookstore_events
// Get upcoming author events at local bookstores
type GetBookstoreEventsInput struct {
    Store     string `json:"store,omitempty"`
    DaysAhead int    `json:"days_ahead,omitempty" jsonschema:"default=30"`
}
```

### Book Community (BookTok/BookTube)

```go
// Tool: get_trending_booktok
// Get trending books from BookTok
type GetTrendingBookTokInput struct {
    Hashtags []string `json:"hashtags,omitempty" jsonschema:"description=Specific hashtags to search"`
    Genre    string   `json:"genre,omitempty"`
    Period   string   `json:"period,omitempty" jsonschema:"enum=day|week|month,default=week"`
}

// Tool: search_booktube
// Search BookTube for book reviews
type SearchBookTubeInput struct {
    BookTitle  string `json:"book_title,omitempty"`
    Author     string `json:"author,omitempty"`
    Channel    string `json:"channel,omitempty" jsonschema:"description=Specific BookTuber channel"`
    ReviewType string `json:"review_type,omitempty" jsonschema:"enum=review|discussion|wrap-up|haul"`
}

// Tool: get_community_sentiment
// Get community sentiment about a book
type GetCommunitySentimentInput struct {
    BookID    string   `json:"book_id,omitempty"`
    ISBN      string   `json:"isbn,omitempty"`
    Sources   []string `json:"sources,omitempty" jsonschema:"description=hardcover|booktok|booktube"`
}
```

### Unified Actions

```go
// Tool: find_book_everywhere
// Search all sources for a book with availability info
type FindBookEverywhereInput struct {
    Query string `json:"query" jsonschema:"required,description=Title, author, or ISBN"`
}

// Returns unified response:
type FindBookEverywhereOutput struct {
    Book         BookMetadata      `json:"book"`
    Hardcover    *HardcoverInfo    `json:"hardcover,omitempty"`
    Library      *LibraryAvail     `json:"library,omitempty"`
    LocalStores  []LocalStoreAvail `json:"local_stores,omitempty"`
    Community    *CommunityInfo    `json:"community,omitempty"`
    Recommendation string          `json:"recommendation"` // "Borrow from APL" etc.
}

// Tool: best_way_to_read
// Determine the best way to access a book
type BestWayToReadInput struct {
    BookID     string   `json:"book_id,omitempty"`
    ISBN       string   `json:"isbn,omitempty"`
    Preferences []string `json:"preferences,omitempty" jsonschema:"description=Priority: library|local|ebook|any"`
}

// Tool: add_to_tbr
// Smart add to TBR - adds to Hardcover + optionally places library hold
type AddToTBRInput struct {
    ISBN          string `json:"isbn,omitempty"`
    Title         string `json:"title,omitempty"`
    Author        string `json:"author,omitempty"`
    PlaceHold     bool   `json:"place_hold,omitempty" jsonschema:"description=Auto-place library hold if available"`
    NotifyInStock bool   `json:"notify_in_stock,omitempty" jsonschema:"description=Notify when available at library"`
}

// Tool: sync_libby_to_hardcover
// Sync current Libby loans to Hardcover as "currently reading"
type SyncLibbyToHardcoverInput struct {
    DryRun bool `json:"dry_run,omitempty" jsonschema:"description=Preview changes without applying"`
}
```

### Knowledge Graph / Semantic

```go
// Tool: get_related_books
// Get semantically related books using Wikidata + embeddings
type GetRelatedBooksInput struct {
    BookID       string `json:"book_id,omitempty"`
    ISBN         string `json:"isbn,omitempty"`
    RelationType string `json:"relation_type,omitempty" jsonschema:"enum=same-author|same-series|similar-themes|influenced-by|same-movement"`
}

// Tool: explore_literary_connection
// Explore connections between books/authors via knowledge graph
type ExploreLiteraryConnectionInput struct {
    FromBook   string `json:"from_book,omitempty"`
    FromAuthor string `json:"from_author,omitempty"`
    ToBook     string `json:"to_book,omitempty"`  
    ToAuthor   string `json:"to_author,omitempty"`
    MaxHops    int    `json:"max_hops,omitempty" jsonschema:"default=3"`
}

// Tool: semantic_book_search (experimental)
// Search books by natural language description
type SemanticBookSearchInput struct {
    Description string `json:"description" jsonschema:"required,description=Natural language description of what you want"`
    // e.g. "Books like Piranesi but with faster pacing and less mystery"
}
```

---

## MCP Resources

Resources expose data that can be read into the LLM context.

```go
// Resource: booklife://library/current
// Current reading list
URI: "booklife://library/current"
Description: "Your currently reading books with progress"
MimeType: "application/json"

// Resource: booklife://library/tbr  
// To-be-read list
URI: "booklife://library/tbr"
Description: "Your want-to-read list from Hardcover"
MimeType: "application/json"

// Resource: booklife://library/recent
// Recently finished books
URI: "booklife://library/recent?days=30"
Description: "Books finished in the last N days"
MimeType: "application/json"

// Resource: booklife://loans
// Current library loans
URI: "booklife://loans"
Description: "Current Libby loans with due dates"
MimeType: "application/json"

// Resource: booklife://holds
// Current library holds
URI: "booklife://holds"  
Description: "Library hold queue with estimated wait times"
MimeType: "application/json"

// Resource: booklife://events
// Upcoming bookstore events
URI: "booklife://events?days=30"
Description: "Author events at Austin bookstores"
MimeType: "application/json"

// Resource: booklife://stats
// Reading statistics
URI: "booklife://stats?year=2025"
Description: "Reading stats for the year"
MimeType: "application/json"

// Resource: booklife://book/{id}
// Detailed book info with all source data
URI: "booklife://book/{hardcover_id}"
Description: "Complete book information from all sources"
MimeType: "application/json"
```

---

## MCP Prompts

Reusable prompt templates for common interactions.

```go
// Prompt: what_should_i_read
// Personalized reading recommendation
Prompt{
    Name: "what_should_i_read",
    Description: "Get a personalized book recommendation",
    Arguments: []PromptArgument{
        {Name: "mood", Description: "Current reading mood", Required: false},
        {Name: "time", Description: "How much time you have", Required: false},
    },
}

// Prompt: book_summary
// Get a comprehensive book summary
Prompt{
    Name: "book_summary",
    Description: "Get a comprehensive summary of a book",
    Arguments: []PromptArgument{
        {Name: "title", Description: "Book title", Required: true},
        {Name: "spoiler_level", Description: "none|light|full", Required: false},
    },
}

// Prompt: reading_wrap_up
// Generate a reading wrap-up for a time period
Prompt{
    Name: "reading_wrap_up",
    Description: "Generate a reading wrap-up",
    Arguments: []PromptArgument{
        {Name: "period", Description: "week|month|year", Required: true},
    },
}

// Prompt: compare_books
// Compare two or more books
Prompt{
    Name: "compare_books",
    Description: "Compare books side by side",
    Arguments: []PromptArgument{
        {Name: "books", Description: "Comma-separated book titles or IDs", Required: true},
    },
}
```

---

## Data Models

### Core Types

```go
package booklife

import "time"

// Book represents a unified book entity across all sources
type Book struct {
    // Universal identifiers
    ID          string            `json:"id"`           // Internal BookLife ID
    ISBN10      string            `json:"isbn10,omitempty"`
    ISBN13      string            `json:"isbn13,omitempty"`
    
    // Cross-platform IDs
    HardcoverID string            `json:"hardcover_id,omitempty"`
    OpenLibID   string            `json:"openlibrary_id,omitempty"`
    WikidataID  string            `json:"wikidata_id,omitempty"`
    OverdriveID string            `json:"overdrive_id,omitempty"`
    
    // Core metadata
    Title       string            `json:"title"`
    Subtitle    string            `json:"subtitle,omitempty"`
    Authors     []Contributor     `json:"authors"`
    
    // Extended metadata
    Publisher       string        `json:"publisher,omitempty"`
    PublishedDate   string        `json:"published_date,omitempty"`
    PageCount       int           `json:"page_count,omitempty"`
    AudioDuration   int           `json:"audio_duration_seconds,omitempty"`
    
    // Classification
    Genres      []string          `json:"genres,omitempty"`
    Subjects    []string          `json:"subjects,omitempty"`
    
    // Series info
    Series      *SeriesInfo       `json:"series,omitempty"`
    
    // Descriptions
    Description string            `json:"description,omitempty"`
    
    // Cover images
    CoverURL    string            `json:"cover_url,omitempty"`
    
    // User-specific data (from Hardcover)
    UserStatus  *UserBookStatus   `json:"user_status,omitempty"`
    
    // Availability
    LibraryAvailability *LibraryAvailability `json:"library_availability,omitempty"`
    
    // Community data
    HardcoverRating float64       `json:"hardcover_rating,omitempty"`
    HardcoverCount  int           `json:"hardcover_rating_count,omitempty"`
}

type Contributor struct {
    Name string `json:"name"`
    Role string `json:"role"` // author, narrator, illustrator, etc.
}

type SeriesInfo struct {
    Name     string  `json:"name"`
    Position float64 `json:"position"` // Supports 1.5 for novellas
    Total    int     `json:"total,omitempty"`
}

type UserBookStatus struct {
    Status      string    `json:"status"` // reading, read, want-to-read, dnf
    Progress    int       `json:"progress,omitempty"` // 0-100
    Rating      float64   `json:"rating,omitempty"`
    Review      string    `json:"review,omitempty"`
    DateStarted *time.Time `json:"date_started,omitempty"`
    DateFinished *time.Time `json:"date_finished,omitempty"`
    DateAdded   time.Time  `json:"date_added"`
}

type LibraryAvailability struct {
    LibraryName   string    `json:"library_name"`
    MediaID       string    `json:"media_id"`
    Formats       []string  `json:"formats"` // ebook, audiobook
    
    // Per-format availability
    EbookAvailable     bool `json:"ebook_available"`
    EbookCopies        int  `json:"ebook_copies"`
    EbookWaitlistSize  int  `json:"ebook_waitlist_size"`
    
    AudiobookAvailable    bool `json:"audiobook_available"`
    AudiobookCopies       int  `json:"audiobook_copies"`
    AudiobookWaitlistSize int  `json:"audiobook_waitlist_size"`
    
    EstimatedWaitDays int  `json:"estimated_wait_days,omitempty"`
}

type LibbyLoan struct {
    ID            string    `json:"id"`
    MediaID       string    `json:"media_id"`
    Title         string    `json:"title"`
    Author        string    `json:"author"`
    CoverURL      string    `json:"cover_url"`
    Format        string    `json:"format"` // ebook, audiobook
    CheckoutDate  time.Time `json:"checkout_date"`
    DueDate       time.Time `json:"due_date"`
    Progress      float64   `json:"progress"` // 0.0-1.0
    IsReturned    bool      `json:"is_returned"`
}

type LibbyHold struct {
    ID                string    `json:"id"`
    MediaID           string    `json:"media_id"`
    Title             string    `json:"title"`
    Author            string    `json:"author"`
    CoverURL          string    `json:"cover_url"`
    Format            string    `json:"format"`
    HoldPlacedDate    time.Time `json:"hold_placed_date"`
    EstimatedWaitDays int       `json:"estimated_wait_days"`
    QueuePosition     int       `json:"queue_position"`
    IsReady           bool      `json:"is_ready"`
    AutoBorrow        bool      `json:"auto_borrow"`
}

type BookstoreEvent struct {
    Store       string    `json:"store"`
    Title       string    `json:"title"`
    Author      string    `json:"author,omitempty"`
    Book        string    `json:"book,omitempty"`
    DateTime    time.Time `json:"datetime"`
    Description string    `json:"description"`
    URL         string    `json:"url"`
    EventType   string    `json:"event_type"` // signing, reading, discussion
}

type TrendingBook struct {
    Book        Book      `json:"book"`
    TrendScore  float64   `json:"trend_score"`
    Hashtags    []string  `json:"hashtags"`
    VideoCount  int       `json:"video_count"`
    Source      string    `json:"source"` // booktok, booktube
}
```

---

## Project Structure

```
booklife-mcp/
├── booklife.kdl              # Configuration file
├── go.mod
├── go.sum
├── cmd/
│   └── booklife/
│       └── main.go           # Entry point
├── internal/
│   ├── config/
│   │   └── config.go         # KDL config loading
│   ├── server/
│   │   ├── server.go         # MCP server setup
│   │   ├── tools.go          # Tool registrations
│   │   ├── resources.go      # Resource handlers
│   │   └── prompts.go        # Prompt templates
│   ├── providers/
│   │   ├── provider.go       # Provider interface
│   │   ├── hardcover/
│   │   │   ├── client.go     # GraphQL client
│   │   │   ├── queries.go    # GraphQL queries
│   │   │   └── types.go      # Hardcover types
│   │   ├── libby/
│   │   │   ├── client.go     # Libby API client
│   │   │   ├── auth.go       # Clone code auth
│   │   │   └── types.go      # Libby types
│   │   ├── openlibrary/
│   │   │   └── client.go     # Open Library client
│   │   ├── wikidata/
│   │   │   └── client.go     # SPARQL client
│   │   ├── youtube/
│   │   │   └── client.go     # YouTube Data API
│   │   └── bookstores/
│   │       └── scraper.go    # Local bookstore scrapers
│   ├── cache/
│   │   ├── sqlite.go         # SQLite cache
│   │   └── embeddings.go     # Embedding index
│   └── models/
│       └── types.go          # Shared data types
└── pkg/
    └── isbn/
        └── isbn.go           # ISBN validation/conversion
```

---

## Implementation Phases

### Phase 1: Core Infrastructure
- [ ] KDL configuration loading
- [ ] MCP server setup with official Go SDK
- [ ] SQLite cache layer
- [ ] Basic tool registration

### Phase 2: Hardcover Integration
- [ ] GraphQL client
- [ ] `search_books` tool
- [ ] `get_my_library` tool  
- [ ] `update_reading_status` tool
- [ ] `add_to_library` tool

### Phase 3: Libby Integration
- [ ] Clone code authentication
- [ ] `search_library` tool
- [ ] `check_availability` tool
- [ ] `get_loans` / `get_holds` tools
- [ ] `place_hold` tool

### Phase 4: Unified Actions
- [ ] `find_book_everywhere` tool
- [ ] `best_way_to_read` tool
- [ ] `sync_libby_to_hardcover` tool

### Phase 5: Community & Discovery
- [ ] YouTube/BookTube integration
- [ ] BookTok integration (if API available)
- [ ] Local bookstore scrapers
- [ ] Event tracking

### Phase 6: Semantic Features (Experimental)
- [ ] Wikidata SPARQL integration
- [ ] Local embedding index
- [ ] `semantic_book_search` tool
- [ ] `explore_literary_connection` tool

---

## Dependencies

```go
// go.mod
module github.com/yourusername/booklife-mcp

go 1.22

require (
    github.com/modelcontextprotocol/go-sdk v0.x.x  // Official MCP SDK
    github.com/sblinch/kdl-go v0.x.x               // KDL config parser
    github.com/hasura/go-graphql-client v0.x.x    // GraphQL for Hardcover
    github.com/mattn/go-sqlite3 v1.14.x           // SQLite cache
    golang.org/x/time v0.x.x                       // Rate limiting
)
```

---

## Environment Variables

```bash
# Required
HARDCOVER_API_KEY=your_hardcover_token

# Optional
YOUTUBE_API_KEY=your_youtube_key
TIKTOK_SCRAPER_API_KEY=your_tiktok_scraper_key

# Development
BOOKLIFE_CONFIG_PATH=./booklife.kdl
BOOKLIFE_CACHE_PATH=~/.booklife/cache.db
BOOKLIFE_LOG_LEVEL=debug
```

---

## Usage with Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "booklife": {
      "command": "/path/to/booklife-mcp",
      "args": ["--config", "/path/to/booklife.kdl"],
      "env": {
        "HARDCOVER_API_KEY": "your_key"
      }
    }
  }
}
```

---

## Future Considerations

### v2 Features
- Reading mood tracking over time
- Automatic TBR prioritization based on library availability
- Integration with Kindle highlights (if API available)
- Book club management
- Reading challenges/goals

### Potential Integrations
- Storygraph (if API becomes available)
- Literal.club
- LibraryThing
- Bookshop.org affiliate links for purchase recommendations
