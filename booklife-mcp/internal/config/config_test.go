package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "booklife.kdl")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSecretRefForms(t *testing.T) {
	t.Setenv("BOOKLIFE_TEST_KEY", "from-env")

	tests := []struct {
		name string
		line string
		want string
	}{
		{"env property", `api-key env="BOOKLIFE_TEST_KEY"`, "from-env"},
		{"positional env reference", `api-key "env=BOOKLIFE_TEST_KEY"`, "from-env"},
		{"literal value", `api-key "literal-key"`, "literal-key"},
		{"unquoted number", `api-key 12345`, "12345"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(writeConfig(t, hardcoverConfig(tt.line)))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := string(cfg.Providers.Hardcover.APIKey); got != tt.want {
				t.Fatalf("APIKey = %q, want %q", got, tt.want)
			}
		})
	}
}

func hardcoverConfig(line string) string {
	return "server {\n name \"booklife\"\n}\nproviders {\n hardcover enabled=true {\n  " + line + "\n }\n}\n"
}

func TestMissingRequiredEnvVarFails(t *testing.T) {
	t.Setenv("BOOKLIFE_TEST_UNSET", "")
	_, err := Load(writeConfig(t, hardcoverConfig(`api-key env="BOOKLIFE_TEST_UNSET"`)))
	if err == nil || !strings.Contains(err.Error(), `required environment variable "BOOKLIFE_TEST_UNSET"`) {
		t.Fatalf("expected a missing variable error, got: %v", err)
	}
}

func TestSecretInEnvNameIsNotEchoed(t *testing.T) {
	const secret = "eyJhbGciOiJIUzI1NiJ9.c2VjcmV0.c2lnbmF0dXJl"
	for _, line := range []string{`api-key env="` + secret + `"`, `api-key "env=` + secret + `"`} {
		_, err := Load(writeConfig(t, hardcoverConfig(line)))
		if err == nil {
			t.Fatalf("%s: expected an error", line)
		}
		if strings.Contains(err.Error(), "eyJ") {
			t.Fatalf("error leaked the value: %v", err)
		}
	}
}

func TestSecretRefRejectsAmbiguousForms(t *testing.T) {
	for _, line := range []string{
		`api-key "literal" env="BOOKLIFE_TEST_KEY"`,
		`api-key "one" "two"`,
		`api-key`,
	} {
		if _, err := Load(writeConfig(t, hardcoverConfig(line))); err == nil {
			t.Errorf("%s: expected an error", line)
		}
	}
}

// The sample configs are what people copy, so they have to load and must
// never carry a real credential.
func TestSampleConfigs(t *testing.T) {
	t.Setenv("HARDCOVER_API_KEY", "test-key")
	jwt := regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}`)

	for _, path := range []string{"../../booklife.kdl.example", "../../../booklife.kdl.example"} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if jwt.Match(data) {
				t.Fatal("sample config contains a token; use env=\"VAR_NAME\"")
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := string(cfg.Providers.Hardcover.APIKey); got != "test-key" {
				t.Fatalf("APIKey = %q, want the value of HARDCOVER_API_KEY", got)
			}
		})
	}
}
