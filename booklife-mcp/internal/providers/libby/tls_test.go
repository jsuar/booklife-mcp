package libby

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
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
	var hit atomic.Bool
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hit.Store(true) }))
	defer srv.Close()

	c := newHTTPClient(srv.URL, nil)
	if _, err := c.Get(srv.URL); err == nil {
		t.Fatal("expected a self-signed certificate to be rejected")
	}
	if hit.Load() {
		t.Fatal("request reached a server with an untrusted certificate")
	}
}

// A trusted certificate must still be rejected when it is for the wrong name.
// httptest's certificate covers example.com and 127.0.0.1, not vandal.libbyapp.com.
func TestConfigRejectsTrustedCertificateForWrongName(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()

	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	addr := srv.Listener.Addr().String()

	pinned := tlsConfig(vandalURL + "/tags")
	pinned.RootCAs = roots
	if conn, err := tls.Dial("tcp", addr, pinned); err == nil {
		conn.Close()
		t.Fatal("expected a certificate for another name to be rejected")
	}

	// Control: the same chain verifies under a name it covers, so the failure
	// above comes from the hostname check rather than from trust.
	control := tlsConfig(thunderURL)
	control.ServerName = "example.com"
	control.RootCAs = roots
	conn, err := tls.Dial("tcp", addr, control)
	if err != nil {
		t.Fatalf("control handshake failed: %v", err)
	}
	conn.Close()
}

func TestTransportsAreSharedPerName(t *testing.T) {
	if transportFor(sentryReadURL+"/a") != transportFor(sentryReadURL+"/b") {
		t.Fatal("expected requests to one host to share a transport")
	}
	if transportFor(sentryReadURL) == transportFor(vandalURL) {
		t.Fatal("expected hosts with different names to use different transports")
	}
	if transportFor(thunderURL).Proxy == nil {
		t.Fatal("expected the default proxy settings to be kept")
	}
}

func TestRedirectsStayOnHost(t *testing.T) {
	first, _ := http.NewRequest("GET", sentryReadURL+"/a", nil)
	via := []*http.Request{first}

	sameHost, _ := http.NewRequest("GET", sentryReadURL+"/b", nil)
	if err := sameHostRedirect(sameHost, via); err != nil {
		t.Fatalf("same-host redirect rejected: %v", err)
	}
	otherHost, _ := http.NewRequest("GET", "https://sentry-read.odrsre.overdrive.com/b", nil)
	if err := sameHostRedirect(otherHost, via); err == nil {
		t.Fatal("expected a cross-host redirect to be rejected")
	}
}

// TestLiveTLS sends unauthenticated requests through the production client to
// the real endpoints. Any HTTP response proves the certificate verified.
func TestLiveTLS(t *testing.T) {
	if os.Getenv("BOOKLIFE_LIVE_TESTS") == "" {
		t.Skip("set BOOKLIFE_LIVE_TESTS=1 to check the live certificates")
	}
	for _, endpoint := range []string{sentryReadURL, vandalURL, thunderURL} {
		t.Run(strings.TrimPrefix(endpoint, "https://"), func(t *testing.T) {
			client := newHTTPClient(endpoint, nil)
			client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
			resp, err := client.Get(endpoint + "/")
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			resp.Body.Close()
		})
	}
}

func TestReadLimited(t *testing.T) {
	if data, err := readLimited(strings.NewReader("12345"), 5); err != nil || string(data) != "12345" {
		t.Fatalf("body at the limit: %q, %v", data, err)
	}
	if _, err := readLimited(strings.NewReader("123456"), 5); err == nil {
		t.Fatal("expected a body over the limit to be rejected")
	}
}
