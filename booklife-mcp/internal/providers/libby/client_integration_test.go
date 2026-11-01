//go:build integration

package libby

import (
	"context"
	"os"
	"testing"
	"time"
)

// Integration tests for Libby client - readonly operations only.
// These tests verify schema and field mappings against the real API.
//
// Run with: go test -tags=integration -v ./internal/providers/libby/
//
// Requires: ~/.config/booklife/libby-identity.json with valid credentials

func skipIfNoCredentials(t *testing.T) *Client {
	t.Helper()

	if !HasSavedIdentity() {
		t.Skip("Skipping integration test: no saved Libby identity found")
	}

	client, err := NewClientFromSavedIdentity() // skip TLS verify for testing
	if err != nil {
		t.Fatalf("Failed to create Libby client: %v", err)
	}

	return client
}

func TestIntegration_LibbyClient_HasLibraries(t *testing.T) {
	client := skipIfNoCredentials(t)

	// Verify identity was loaded and libraries synced
	if len(client.libraries) == 0 {
		t.Fatal("Expected at least one library to be linked")
	}

	lib := client.libraries[0]

	// Verify library fields are populated
	if lib.ID == "" {
		t.Error("Library ID should not be empty")
	}
	if lib.Name == "" {
		t.Error("Library Name should not be empty")
	}

	t.Logf("Connected to library: %s", lib.Name)
	t.Logf("  ID (websiteId): %s", lib.ID)
	t.Logf("  CardID: %s", lib.CardID)
	t.Logf("  AdvantageKey: %s", lib.Advantagekey)
}

func TestIntegration_LibbyClient_Search(t *testing.T) {
	client := skipIfNoCredentials(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Search for a well-known book
	books, total, err := client.Search(ctx, "Project Hail Mary", nil, false, 0, 5)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if total == 0 {
		t.Skip("No results found - library may not have this book")
	}

	// Verify at least one result
	if len(books) == 0 {
		t.Fatal("Expected at least one book result")
	}

	book := books[0]

	// Verify Book model fields are mapped correctly
	t.Run("Book fields populated", func(t *testing.T) {
		if book.Title == "" {
			t.Error("Book.Title should not be empty")
		}
		if book.OverdriveID == "" {
			t.Error("Book.OverdriveID should not be empty (this is the media ID)")
		}
		// Authors may be empty for some entries, but log it
		if len(book.Authors) == 0 {
			t.Log("Warning: Book.Authors is empty")
		} else if book.Authors[0].Name == "" {
			t.Error("Book.Authors[0].Name should not be empty when authors exist")
		}
	})

	// Verify LibraryAvailability is attached
	t.Run("LibraryAvailability populated", func(t *testing.T) {
		if book.LibraryAvailability == nil {
			t.Fatal("Book.LibraryAvailability should not be nil")
		}

		avail := book.LibraryAvailability
		if avail.MediaID == "" {
			t.Error("LibraryAvailability.MediaID should not be empty")
		}
		if avail.LibraryName == "" {
			t.Error("LibraryAvailability.LibraryName should not be empty")
		}
		// At least one format should be present
		if len(avail.Formats) == 0 && !avail.EbookAvailable && !avail.AudiobookAvailable {
			t.Log("Warning: No formats indicated as available")
		}
	})

	t.Logf("Search returned %d results (total: %d)", len(books), total)
	t.Logf("First result: %s by %v (MediaID: %s)", book.Title, book.Authors, book.OverdriveID)
}

func TestIntegration_LibbyClient_CheckAvailability(t *testing.T) {
	client := skipIfNoCredentials(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check a book by title
	avail, err := client.CheckAvailability(ctx, "", "Dune", "Frank Herbert")
	if err != nil {
		t.Fatalf("CheckAvailability failed: %v", err)
	}

	if avail == nil {
		t.Skip("Book not found in library catalog")
	}

	// Verify LibraryAvailability fields
	if avail.MediaID == "" {
		t.Error("MediaID should not be empty")
	}
	if avail.LibraryName == "" {
		t.Error("LibraryName should not be empty")
	}

	t.Logf("Availability: MediaID=%s, Ebook=%v, Audiobook=%v, Wait=%d days",
		avail.MediaID, avail.EbookAvailable, avail.AudiobookAvailable, avail.EstimatedWaitDays)
}

func TestIntegration_LibbyClient_GetLoans(t *testing.T) {
	client := skipIfNoCredentials(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	loans, err := client.GetLoans(ctx)
	if err != nil {
		t.Fatalf("GetLoans failed: %v", err)
	}

	// Loans may be empty if nothing is checked out
	t.Logf("Found %d active loans", len(loans))

	if len(loans) > 0 {
		loan := loans[0]

		// Verify LibbyLoan fields
		if loan.ID == "" {
			t.Error("LibbyLoan.ID should not be empty")
		}
		if loan.MediaID == "" {
			t.Error("LibbyLoan.MediaID should not be empty")
		}
		if loan.Title == "" {
			t.Error("LibbyLoan.Title should not be empty")
		}
		if loan.Format == "" {
			t.Log("Warning: LibbyLoan.Format is empty")
		}
		if loan.DueDate.IsZero() {
			t.Log("Warning: LibbyLoan.DueDate is zero")
		}

		t.Logf("Loan: %s (Format: %s, Due: %s)", loan.Title, loan.Format, loan.DueDate.Format("2006-01-02"))
	}
}

func TestIntegration_LibbyClient_GetHolds(t *testing.T) {
	client := skipIfNoCredentials(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	holds, err := client.GetHolds(ctx)
	if err != nil {
		t.Fatalf("GetHolds failed: %v", err)
	}

	// Holds may be empty
	t.Logf("Found %d active holds", len(holds))

	if len(holds) > 0 {
		hold := holds[0]

		// Verify LibbyHold fields
		if hold.ID == "" {
			t.Error("LibbyHold.ID should not be empty")
		}
		if hold.MediaID == "" {
			t.Error("LibbyHold.MediaID should not be empty")
		}
		if hold.Title == "" {
			t.Error("LibbyHold.Title should not be empty")
		}
		// QueuePosition should be >= 0
		if hold.QueuePosition < 0 {
			t.Errorf("LibbyHold.QueuePosition should be >= 0, got %d", hold.QueuePosition)
		}

		t.Logf("Hold: %s (Position: #%d, Ready: %v, Wait: %d days)",
			hold.Title, hold.QueuePosition, hold.IsReady, hold.EstimatedWaitDays)
	}
}

func TestIntegration_LibbyClient_GetTags(t *testing.T) {
	client := skipIfNoCredentials(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tags, err := client.GetTags(ctx)
	if err != nil {
		t.Fatalf("GetTags failed: %v", err)
	}

	// Tags may be empty if user hasn't tagged anything
	t.Logf("Found %d tags", len(tags))

	for tag, mediaIDs := range tags {
		if tag == "" {
			t.Error("Tag name should not be empty")
		}
		t.Logf("Tag '%s': %d books", tag, len(mediaIDs))
	}
}

// TestIntegration_LibbyClient_IdentityPersistence verifies identity save/load
func TestIntegration_LibbyClient_IdentityPersistence(t *testing.T) {
	if !HasSavedIdentity() {
		t.Skip("Skipping: no saved identity")
	}

	identity, err := LoadIdentity()
	if err != nil {
		t.Fatalf("LoadIdentity failed: %v", err)
	}

	// Verify identity fields
	if identity.ChipKey == "" {
		t.Error("Identity.ChipKey should not be empty")
	}

	// Verify the identity file exists at expected path
	path, err := identityPath()
	if err != nil {
		t.Fatalf("identityPath failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Identity file should exist at %s", path)
	}

	t.Logf("Identity loaded from %s", path)
}
