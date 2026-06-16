package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestWalker(t *testing.T) {
	tmp, err := os.MkdirTemp("", "prowiki-scanner-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	os.WriteFile(filepath.Join(tmp, ".gitignore"), []byte("ignored.txt\nnode_modules/"), 0644)
	os.WriteFile(filepath.Join(tmp, "kept.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(tmp, "ignored.txt"), []byte("bye"), 0644)
	
	os.Mkdir(filepath.Join(tmp, "node_modules"), 0755)
	os.WriteFile(filepath.Join(tmp, "node_modules", "lib.js"), []byte("var x = 1;"), 0644)

	walker := NewDefaultWalker()
	
	var found []string
	
	ctx := context.Background()
	out, errc := walker.Walk(ctx, tmp, domain.WalkOptions{})

	for file := range out {
		found = append(found, filepath.Base(file.Path))
	}

	if err := <-errc; err != nil {
		t.Fatal(err)
	}

	if len(found) != 2 { // .gitignore, kept.txt
		t.Fatalf("expected 2 files, got %v", found)
	}

	for _, name := range found {
		if name == "ignored.txt" || name == "lib.js" {
			t.Errorf("found ignored file: %s", name)
		}
	}
}
