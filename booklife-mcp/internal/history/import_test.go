package history

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const validTimelineURL = "https://share.libbyapp.com/data/abc/libbytimeline-all-loans.json"

func validate(t *testing.T, rawURL string) error {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	return validateTimelineURL(u)
}

func TestValidateTimelineURL(t *testing.T) {
	accepted := []string{
		validTimelineURL,
		"https://Share.LibbyApp.com/data/abc/libbytimeline-all-loans.json",
		"https://share.libbyapp.com:443/data/abc/libbytimeline-all-loans.json",
	}
	for _, rawURL := range accepted {
		if err := validate(t, rawURL); err != nil {
			t.Errorf("expected %q to be accepted: %v", rawURL, err)
		}
	}

	rejected := []string{
		"http://share.libbyapp.com/data/abc.json",
		"https://share.libbyapp.com.evil.example/data/abc.json",
		"https://evil.example/share.libbyapp.com/data/abc.json",
		"https://share.libbyapp.com@evil.example/data/abc.json",
		"https://user@share.libbyapp.com/data/abc.json",
		"https://share.libbyapp.com:8443/data/abc.json",
		"https://share.libbyapp.com/",
		"https://share.libbyapp.com/admin.json",
		"https://share.libbyapp.com/data/abc",
		"http://127.0.0.1:8080/data/abc.json",
		"http://169.254.169.254/latest/meta-data/",
		"https://localhost/data/abc.json",
		"file:///data/abc.json",
		"",
	}
	for _, rawURL := range rejected {
		if err := validate(t, rawURL); err == nil {
			t.Errorf("expected %q to be rejected", rawURL)
		}
	}
}

// roundTripFunc records requests instead of using the network.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func response(status int, body string, header http.Header) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestFetchRejectsBeforeAnyRequest(t *testing.T) {
	client := newTimelineClient()
	client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("no request should be sent")
		return nil, nil
	})
	if _, err := fetchTimeline(context.Background(), client, "http://127.0.0.1:1/"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestFetchFollowsNoUnsafeRedirect(t *testing.T) {
	const target = "http://10.0.0.5/admin?token=abc"
	requests := 0
	client := newTimelineClient()
	client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return response(http.StatusFound, "", http.Header{"Location": {target}}), nil
	})

	_, err := fetchTimeline(context.Background(), client, validTimelineURL)
	if err == nil {
		t.Fatal("expected the redirect to be rejected")
	}
	if requests != 1 {
		t.Fatalf("sent %d requests, want 1", requests)
	}
	if strings.Contains(err.Error(), "10.0.0.5") {
		t.Fatalf("error leaked the redirect target: %v", err)
	}
}

func TestFetchFollowsSafeRedirect(t *testing.T) {
	next := "https://share.libbyapp.com/data/def/libbytimeline-all-loans.json"
	client := newTimelineClient()
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() == next {
			return response(http.StatusOK, `{"version":1,"timeline":[]}`, nil), nil
		}
		return response(http.StatusFound, "", http.Header{"Location": {next}}), nil
	})
	if _, err := fetchTimeline(context.Background(), client, validTimelineURL); err != nil {
		t.Fatalf("redirect within the allowed host failed: %v", err)
	}
}

func TestFetchErrorOmitsShareURL(t *testing.T) {
	client := newTimelineClient()
	client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection reset")
	})
	_, err := fetchTimeline(context.Background(), client, validTimelineURL)
	if err == nil || strings.Contains(err.Error(), "/data/abc") {
		t.Fatalf("expected an error without the share URL, got: %v", err)
	}
}

func TestFetchHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := newTimelineClient()
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, req.Context().Err()
	})
	_, err := fetchTimeline(ctx, client, validTimelineURL)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestReadTimelineErrorOmitsBody(t *testing.T) {
	_, err := readTimeline(response(http.StatusForbidden, "internal-secret-page", nil), 1024)
	if err == nil || strings.Contains(err.Error(), "internal-secret-page") {
		t.Fatalf("expected an error without the body, got: %v", err)
	}
}

func TestReadTimelineSizeLimit(t *testing.T) {
	body := `{"version":1,"timeline":[]}`
	size := int64(len(body))

	if _, err := readTimeline(response(http.StatusOK, body, nil), size); err != nil {
		t.Fatalf("body at the limit rejected: %v", err)
	}

	_, err := readTimeline(response(http.StatusOK, body, nil), size-1)
	if err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("expected a size error, got: %v", err)
	}

	declared := response(http.StatusOK, "", nil)
	declared.ContentLength = size + 1
	_, err = readTimeline(declared, size)
	if err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("expected a size error from Content-Length, got: %v", err)
	}
}
