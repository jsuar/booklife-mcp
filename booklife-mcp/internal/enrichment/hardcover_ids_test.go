package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetByTitleAuthorDecodesIDs(t *testing.T) {
	for name, ids := range map[string]string{
		"numbers": `[123, 456]`,
		"strings": `["123", "456"]`,
	} {
		t.Run(name, func(t *testing.T) {
			var lookup map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if strings.Contains(string(body), "search(") {
					fmt.Fprintf(w, `{"data":{"search":{"ids":%s}}}`, ids)
					return
				}
				json.Unmarshal(body, &lookup)
				fmt.Fprint(w, `{"data":{"books":[{"id":123,"title":"Dune"}]}}`)
			}))
			defer srv.Close()

			enricher := NewHardcoverEnricherDirect(srv.URL, "test-key")
			if _, err := enricher.GetByTitleAuthor(context.Background(), "Dune", "Frank Herbert"); err != nil {
				t.Fatalf("GetByTitleAuthor: %v", err)
			}
			if lookup == nil {
				t.Fatal("expected a follow-up lookup by ID")
			}
		})
	}
}
