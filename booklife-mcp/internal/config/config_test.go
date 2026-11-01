package config

import (
	"os"
	"path/filepath"
	"regexp"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(writeConfig(t, "server {\n name \"booklife\"\n}\nproviders {\n hardcover enabled=true {\n  "+tt.line+"\n }\n}\n"))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := string(cfg.Providers.Hardcover.APIKey); got != tt.want {
				t.Fatalf("APIKey = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMissingRequiredEnvVarFails(t *testing.T) {
	_, err := Load(writeConfig(t, "server {\n name \"booklife\"\n}\nproviders {\n hardcover enabled=true {\n  api-key env=\"BOOKLIFE_TEST_UNSET\"\n }\n}\n"))
	if err == nil {
		t.Fatal("expected an error when the required variable is unset")
	}
}

// The sample configs are what people copy, so they have to load and must
// never carry a real credential.
func TestSampleConfigs(t *testing.T) {
	t.Setenv("HARDCOVER_API_KEY", "test-key")
	jwt := regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}`)

	for _, path := range []string{"../../booklife.kdl", "../../../booklife.kdl"} {
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
