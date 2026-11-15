package serve

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kite-plus/kite/internal/api"
)

// Every regular file counts, wherever it sits under the directory measured.
func TestDiskUsageAddsEveryFileUnderIt(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "posts", "one")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "keeper.bin"), make([]byte, 10), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := diskUsage(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if want := (api.Storage{Content: 15, Files: 2}); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// A project with nothing in it yet holds nothing; it is not an error.
func TestDiskUsageOfAMissingDirectoryIsNothing(t *testing.T) {
	got, err := diskUsage(context.Background(), filepath.Join(t.TempDir(), "content"))
	if err != nil {
		t.Fatal(err)
	}
	if got != (api.Storage{}) {
		t.Errorf("got %+v, want nothing", got)
	}
}

// A cancelled request stops the walk rather than finishing it.
func TestDiskUsageStopsWhenTheRequestDoes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := diskUsage(ctx, root); err == nil {
		t.Error("a cancelled request measured the whole project")
	}
}
