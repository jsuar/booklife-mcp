package history

import (
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

// Databases written by earlier versions already contain NULLs.
func TestNewStoreRepairsExistingNulls(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.db.Exec(`
		INSERT INTO history (title_id, title, author, publisher, isbn, timestamp, activity,
			details, library, library_key, format, cover_url, color)
		VALUES ('1', 'Old Loan', 'An Author', NULL, NULL, 1, 'Borrowed',
			NULL, 'Library', '', 'ebook', NULL, NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ { // the repair must be safe to run on every start
		store, err = NewStore(dataDir)
		if err != nil {
			t.Fatal(err)
		}
		entries, _, err := store.GetHistory(0, 10)
		store.Close()
		if err != nil {
			t.Fatalf("GetHistory: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
	}
}
