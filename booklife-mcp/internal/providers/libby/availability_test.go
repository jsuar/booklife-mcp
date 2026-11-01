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
	}, "")
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
	}, "")
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
	}, "")
	if got.MediaID != "second" || !got.EbookAvailable {
		t.Fatalf("got %+v, want the first available edition", got)
	}
}

func TestMergeEditionsFallsBackToAudiobook(t *testing.T) {
	got := mergeEditions([]models.Book{
		ebook("ebook-1", "Dune", "Frank Herbert", false),
		audiobook("audio-1", "Dune", "Frank Herbert", true),
	}, "")
	if got.MediaID != "audio-1" {
		t.Fatalf("MediaID = %q, want the available audiobook", got.MediaID)
	}
}

func TestMergeEditionsHandlesMissingData(t *testing.T) {
	if got := mergeEditions(nil, ""); got != nil {
		t.Fatalf("no results: got %+v", got)
	}
	if got := mergeEditions([]models.Book{{Title: "Dune"}}, ""); got != nil {
		t.Fatalf("no availability: got %+v", got)
	}
	got := mergeEditions([]models.Book{
		{Title: "Dune"},
		ebook("ebook-1", "Dune", "Frank Herbert", true),
	}, "")
	if got == nil || got.MediaID != "ebook-1" {
		t.Fatalf("got %+v, want the first result with availability", got)
	}
}

func TestMergeEditionsKeepsDifferentBooksApart(t *testing.T) {
	cases := map[string][]models.Book{
		"shared series prefix": {
			ebook("first", "Sherlock Holmes: The Hound of the Baskervilles", "Arthur Conan Doyle", false),
			ebook("other", "Sherlock Holmes: A Study in Scarlet", "Arthur Conan Doyle", true),
		},
		"adaptation": {
			ebook("first", "Dune", "Frank Herbert", false),
			ebook("other", "Dune: The Graphic Novel", "Frank Herbert", true),
		},
		"no comparable title": {
			ebook("first", "(?)", "Frank Herbert", false),
			ebook("other", "(!)", "Frank Herbert", true),
		},
		"no author": {
			ebook("first", "Dune", "", false),
			ebook("other", "Dune", "", true),
		},
	}
	for name, books := range cases {
		t.Run(name, func(t *testing.T) {
			got := mergeEditions(books, "")
			if got.MediaID != "first" || got.EbookAvailable {
				t.Fatalf("merged a different book: %+v", got)
			}
		})
	}
}

func TestMergeEditionsAnchorsOnRequestedTitle(t *testing.T) {
	got := mergeEditions([]models.Book{
		ebook("messiah", "Dune Messiah", "Frank Herbert", true),
		ebook("dune", "Dune", "Frank Herbert", false),
		audiobook("dune-audio", "Dune [Unabridged]", "Frank Herbert", true),
	}, "Dune")
	if got.EbookAvailable {
		t.Fatalf("took the sequel's availability: %+v", got)
	}
	if got.MediaID != "dune-audio" {
		t.Fatalf("MediaID = %q, want the requested book's audiobook", got.MediaID)
	}
}

func TestMergeEditionsReportsMediaIDPerFormat(t *testing.T) {
	got := mergeEditions([]models.Book{
		ebook("ebook-1", "Dune", "Frank Herbert", false),
		audiobook("audio-1", "Dune - Unabridged", "Frank Herbert", true),
	}, "")
	if got.EbookMediaID != "ebook-1" || got.AudiobookMediaID != "audio-1" {
		t.Fatalf("got %+v", got)
	}
}

func TestMergeEditionsCountsAnEditionOnce(t *testing.T) {
	book := ebook("ebook-1", "Dune", "Frank Herbert", true)
	got := mergeEditions([]models.Book{book, book}, "")
	if len(got.Formats) != 1 {
		t.Fatalf("Formats = %v", got.Formats)
	}
}

func TestMergeEditionsClearsWaitWhenAvailable(t *testing.T) {
	waitlisted := audiobook("audio-1", "Dune", "Frank Herbert", false)
	waitlisted.LibraryAvailability.EstimatedWaitDays = 30
	got := mergeEditions([]models.Book{waitlisted, ebook("ebook-1", "Dune", "Frank Herbert", true)}, "")
	if got.EstimatedWaitDays != 0 {
		t.Fatalf("EstimatedWaitDays = %d, want 0", got.EstimatedWaitDays)
	}
}
