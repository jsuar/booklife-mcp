package dirs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func checkMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != want {
		t.Errorf("%s mode = %o, want %o", path, info.Mode().Perm(), want)
	}
}

func TestSecureStoreTightensExistingFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file modes")
	}
	dataDir := filepath.Join(t.TempDir(), "data")
	if err := os.Mkdir(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dataDir, "history.db")
	for _, path := range []string{dbPath, dbPath + "-wal"} {
		if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := SecureStore(dataDir, dbPath); err != nil {
		t.Fatal(err)
	}

	checkMode(t, dataDir, 0700)
	checkMode(t, dbPath, 0600)
	checkMode(t, dbPath+"-wal", 0600)
	if data, err := os.ReadFile(dbPath); err != nil || string(data) != "data" {
		t.Fatalf("contents changed: %q, %v", data, err)
	}
}

func TestSecureStoreCreatesPrivateDatabase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file modes")
	}
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "new.db")
	if err := SecureStore(dataDir, dbPath); err != nil {
		t.Fatal(err)
	}
	checkMode(t, dbPath, 0600)
}

func TestSecureStoreReportsFailureWithoutPanicking(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if err := SecureStore(missing, filepath.Join(missing, "x.db")); err == nil {
		t.Fatal("expected an error for a missing directory")
	}
}
