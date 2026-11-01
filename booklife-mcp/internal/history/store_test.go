package history

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/user/booklife-mcp/internal/models"
)

func TestNewStoreIsPrivate(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	store, err := NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for path, want := range map[string]os.FileMode{
		dataDir:                              0700,
		filepath.Join(dataDir, "history.db"): 0600,
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != want {
			t.Errorf("%s mode = %o, want %o", path, info.Mode().Perm(), want)
		}
	}
}

func TestCurrentLoanLeavesHistoryReadable(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	loan := models.LibbyLoan{
		ID:           "123",
		Title:        "A Loan",
		Author:       "An Author",
		Format:       "ebook",
		CheckoutDate: time.Now().Add(-24 * time.Hour),
		DueDate:      time.Now().Add(24 * time.Hour),
	}
	if err := store.ImportCurrentLoan(loan); err != nil {
		t.Fatal(err)
	}

	entries, total, err := store.GetHistory(0, 10)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if total != 1 || len(entries) != 1 || entries[0].Title != "A Loan" {
		t.Fatalf("got total %d, entries %+v", total, entries)
	}
	if _, _, err := store.SearchHistory("Loan", 0, 10); err != nil {
		t.Fatalf("SearchHistory: %v", err)
	}
}

// legacyHistoryTable is the schema from before the columns had defaults.
const legacyHistoryTable = `CREATE TABLE history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title_id TEXT NOT NULL,
	title TEXT NOT NULL,
	author TEXT NOT NULL,
	publisher TEXT,
	isbn TEXT,
	timestamp INTEGER NOT NULL,
	activity TEXT NOT NULL,
	details TEXT,
	library TEXT NOT NULL,
	library_key TEXT NOT NULL,
	format TEXT NOT NULL,
	cover_url TEXT,
	color TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(title_id, timestamp, activity)
)`

func TestNewStoreRepairsLegacyDatabase(t *testing.T) {
	dataDir := t.TempDir()
	db, err := sql.Open("sqlite3", filepath.Join(dataDir, "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacyHistoryTable); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO history (title_id, title, author, timestamp, activity, library, library_key, format)
		VALUES ('1', 'Old Loan', 'An Author', 1, 'Returned', 'Library', '', 'ebook')`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	for i := 0; i < 2; i++ { // the repair must be safe to run on every start
		store, err := NewStore(dataDir)
		if err != nil {
			t.Fatal(err)
		}
		entries, _, err := store.GetHistory(0, 10)
		if err != nil {
			t.Fatalf("GetHistory: %v", err)
		}
		if len(entries) != 1 || entries[0].Publisher != "" || entries[0].ISBN != "" {
			t.Fatalf("got %+v", entries)
		}
		if _, _, err := store.SearchHistory("Old", 0, 10); err != nil {
			t.Fatalf("SearchHistory: %v", err)
		}
		if _, err := store.GetUnsyncedReturns("hardcover"); err != nil {
			t.Fatalf("GetUnsyncedReturns: %v", err)
		}
		store.Close()
	}
}

// Another BookLife process may hold the write lock during startup.
func TestNewStoreStartsWhileDatabaseIsLocked(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()

	db, err := sql.Open("sqlite3", filepath.Join(dataDir, "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")

	store, err = NewStore(dataDir)
	if err != nil {
		t.Fatalf("NewStore with a held write lock: %v", err)
	}
	store.Close()
}

func TestCurrentLoanKeepsExistingMetadataAndID(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	checkout := time.UnixMilli(1700000000000)
	_, err = store.db.Exec(`INSERT INTO history (title_id, title, author, publisher, isbn, timestamp, activity,
			details, library, library_key, format, cover_url, color)
		VALUES ('123', 'A Loan', 'An Author', 'A Publisher', '9780000000000', ?, 'Borrowed',
			'', 'My Library', 'lib', 'ebook', '', '')`, checkout.UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	var before int64
	if err := store.db.QueryRow(`SELECT id FROM history`).Scan(&before); err != nil {
		t.Fatal(err)
	}

	loan := models.LibbyLoan{ID: "123", Title: "A Loan", Author: "An Author", Format: "ebook",
		CheckoutDate: checkout, DueDate: time.Now().Add(72 * time.Hour)}
	if err := store.ImportCurrentLoan(loan); err != nil {
		t.Fatal(err)
	}

	var after int64
	var publisher, library, details string
	err = store.db.QueryRow(`SELECT id, publisher, library, details FROM history`).Scan(&after, &publisher, &library, &details)
	if err != nil {
		t.Fatal(err)
	}
	if after != before || publisher != "A Publisher" || library != "My Library" {
		t.Fatalf("row changed: id %d->%d, publisher %q, library %q", before, after, publisher, library)
	}
	if details == "" {
		t.Fatal("expected the loan details to be refreshed")
	}
}
