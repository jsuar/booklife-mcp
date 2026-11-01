package hardcover

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchBooksDecodesIDs(t *testing.T) {
	for name, ids := range map[string]string{
		"numbers": `[123, 456]`,
		"strings": `["123", "456"]`,
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintf(w, `{"data":{"search":{"ids":%s,"results":{"hits":[
					{"document":{"id":"123","title":"Dune","author_names":["Frank Herbert"]}}
				]}}}}`, ids)
			}))
			defer srv.Close()

			client, err := NewClient(srv.URL, "test-key")
			if err != nil {
				t.Fatal(err)
			}
			books, total, err := client.SearchBooks(context.Background(), "dune", 0, 10)
			if err != nil {
				t.Fatalf("SearchBooks: %v", err)
			}
			if total != 2 || len(books) != 1 || books[0].Title != "Dune" {
				t.Fatalf("got total %d, books %+v", total, books)
			}
		})
	}
}
