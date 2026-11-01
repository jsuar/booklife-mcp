package libby

import (
	"testing"

	"github.com/user/booklife-mcp/internal/models"
)

func ebook(id, title, author string, available bool) models.Book {
	return models.Book{
		Title:   title,
		Authors: []models.Contributor{{Name: author}},
		LibraryAvailability: &models.LibraryAvailability{
			MediaID:        id,
			Formats:        []string{"ebook"},
			EbookAvailable: available,
			EbookCopies:    3,
		},
	}
}

func audiobook(id, title, author string, available bool) models.Book {
	return models.Book{
		Title:   title,
		Authors: []models.Contributor{{Name: author}},
		LibraryAvailability: &models.LibraryAvailability{
			MediaID:            id,
			Formats:            []string{"audiobook"},
			AudiobookAvailable: available,
			AudiobookCopies:    2,
		},
	}
}

func TestMergeEditionsFindsEbookBehindAudiobook(t *testing.T) {
	got := mergeEditions([]models.Book{
		audiobook("audio-1", "Dune (Unabridged)", "Frank Herbert", false),
		ebook("ebook-1", "Dune", "Frank Herbert", true),
	})
	if !got.EbookAvailable {
		t.Fatal("expected the ebook to be available")
	}
	if got.MediaID != "ebook-1" {
		t.Fatalf("MediaID = %q, want the available ebook", got.MediaID)
	}
	if len(got.Formats) != 2 {
		t.Fatalf("Formats = %v, want both", got.Formats)
	}
}

func TestMergeEditionsIgnoresOtherBooks(t *testing.T) {
	got := mergeEditions([]models.Book{
		ebook("dune", "Dune", "Frank Herbert", false),
		ebook("messiah", "Dune Messiah", "Frank Herbert", true),
		audiobook("other", "Dune", "Someone Else", true),
	})
	if got.EbookAvailable || got.AudiobookAvailable {
		t.Fatalf("availability taken from another book: %+v", got)
	}
	if got.MediaID != "dune" {
		t.Fatalf("MediaID = %q, want the top result", got.MediaID)
	}
}

func TestMergeEditionsPrefersAvailableEdition(t *testing.T) {
	got := mergeEditions([]models.Book{
		ebook("first", "Dune", "Frank Herbert", false),
		ebook("second", "Dune", "Frank Herbert", true),
		ebook("third", "Dune", "Frank Herbert", true),
	})
	if got.MediaID != "second" || !got.EbookAvailable {
		t.Fatalf("got %+v, want the first available edition", got)
	}
}

func TestMergeEditionsFallsBackToAudiobook(t *testing.T) {
	got := mergeEditions([]models.Book{
		ebook("ebook-1", "Dune", "Frank Herbert", false),
		audiobook("audio-1", "Dune", "Frank Herbert", true),
	})
	if got.MediaID != "audio-1" {
		t.Fatalf("MediaID = %q, want the available audiobook", got.MediaID)
	}
}

func TestMergeEditionsHandlesMissingData(t *testing.T) {
	if got := mergeEditions(nil); got != nil {
		t.Fatalf("no results: got %+v", got)
	}
	if got := mergeEditions([]models.Book{{Title: "Dune"}}); got != nil {
		t.Fatalf("no availability: got %+v", got)
	}
	got := mergeEditions([]models.Book{
		{Title: "Dune"},
		ebook("ebook-1", "Dune", "", true),
	})
	if got == nil || got.MediaID != "ebook-1" {
		t.Fatalf("got %+v, want the first result with availability", got)
	}
}
