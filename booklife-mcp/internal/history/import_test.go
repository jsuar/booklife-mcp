package history

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestValidateTimelineURL(t *testing.T) {
	valid := "https://share.libbyapp.com/data/abc/libbytimeline-all-loans.json"
	if err := ValidateTimelineURL(valid); err != nil {
		t.Fatalf("valid URL rejected: %v", err)
	}

	rejected := []string{
		"http://share.libbyapp.com/data/abc.json",
		"https://share.libbyapp.com.evil.example/data/abc.json",
		"https://evil.example/share.libbyapp.com/",
		"https://share.libbyapp.com@evil.example/",
		"https://user@share.libbyapp.com/",
		"https://share.libbyapp.com:8443/",
		"http://127.0.0.1:8080/admin",
		"http://169.254.169.254/latest/meta-data/",
		"https://localhost/",
		"file:///etc/passwd",
		"://bad",
		"",
	}
	for _, rawURL := range rejected {
		if err := ValidateTimelineURL(rawURL); err == nil {
			t.Errorf("expected %q to be rejected", rawURL)
		}
	}
}

func TestFetchTimelineRejectsBeforeAnyRequest(t *testing.T) {
	im := &Importer{}
	if _, err := im.FetchTimeline("http://127.0.0.1:1/"); err == nil ||
		strings.Contains(err.Error(), "fetching timeline") {
		t.Fatalf("expected a validation error, got: %v", err)
	}
}

func TestRedirectsAreValidated(t *testing.T) {
	via := make([]*http.Request, 1)
	req, _ := http.NewRequest("GET", "http://127.0.0.1:8080/", nil)
	if err := checkTimelineRedirect(req, via); err == nil {
		t.Fatal("expected a redirect to a local address to be rejected")
	}
	req, _ = http.NewRequest("GET", "https://share.libbyapp.com/next", nil)
	if err := checkTimelineRedirect(req, via); err != nil {
		t.Fatalf("redirect within the allowed host rejected: %v", err)
	}
	if err := checkTimelineRedirect(req, make([]*http.Request, 5)); err == nil {
		t.Fatal("expected a redirect limit")
	}
}

func TestReadTimelineErrorOmitsBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader("internal-secret-page")),
	}
	_, err := readTimeline(resp)
	if err == nil || strings.Contains(err.Error(), "internal-secret-page") {
		t.Fatalf("expected an error without the body, got: %v", err)
	}
}
