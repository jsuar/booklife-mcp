package tbr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStoreIsPrivate(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	store, err := NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	for path, want := range map[string]os.FileMode{
		dataDir:                          0700,
		filepath.Join(dataDir, "tbr.db"): 0600,
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != want {
			t.Errorf("%s mode = %o, want %o", path, info.Mode().Perm(), want)
		}
	}
}
