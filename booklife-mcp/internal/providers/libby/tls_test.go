package libby

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTLSServerName(t *testing.T) {
	tests := []struct {
		endpoint string
		want     string
	}{
		{sentryReadURL + "/chip/sync", "sentry-read.odrsre.overdrive.com"},
		{sentryReadURL, "sentry-read.odrsre.overdrive.com"},
		{vandalURL + "/tags", "vandal.libbyapp.com"},
		{thunderURL + "/v2/libraries/x/media", ""},
		// A lookalike host must not inherit another endpoint's name.
		{sentryReadURL + ".evil.example/x", ""},
		{"https://example.com/", ""},
	}
	for _, tt := range tests {
		if got := tlsServerName(tt.endpoint); got != tt.want {
			t.Errorf("tlsServerName(%q) = %q, want %q", tt.endpoint, got, tt.want)
		}
	}
}

func TestClientRejectsUntrustedCertificate(t *testing.T) {
	var hit bool
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hit = true }))
	defer srv.Close()

	c := newHTTPClient(srv.URL, nil)
	if _, err := c.Get(srv.URL); err == nil {
		t.Fatal("expected a self-signed certificate to be rejected")
	}
	if hit {
		t.Fatal("request reached a server with an untrusted certificate")
	}
}

// TestLiveTLS dials the real endpoints with Go's verifier, without credentials.
func TestLiveTLS(t *testing.T) {
	if os.Getenv("BOOKLIFE_LIVE_TESTS") == "" {
		t.Skip("set BOOKLIFE_LIVE_TESTS=1 to check the live certificates")
	}
	for _, endpoint := range []string{sentryReadURL, vandalURL, thunderURL} {
		t.Run(strings.TrimPrefix(endpoint, "https://"), func(t *testing.T) {
			u, err := url.Parse(endpoint)
			if err != nil {
				t.Fatal(err)
			}
			name := tlsServerName(endpoint)
			if name == "" {
				name = u.Hostname()
			}
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			conn, err := tls.DialWithDialer(dialer, "tcp", u.Hostname()+":443", &tls.Config{ServerName: name})
			if err != nil {
				t.Fatalf("verifying %s as %s: %v", u.Hostname(), name, err)
			}
			conn.Close()
		})
	}
}
