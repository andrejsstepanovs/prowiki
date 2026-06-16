package ast

import (
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

func TestHeuristicParser(t *testing.T) {
	parser := NewHeuristicParser()

	content1 := []byte(`
package main

// This is a comment
func main() {
	println("hello")
}
`)

	content2 := []byte(`
package main

// This is a DIFFERENT comment
// With an extra line
func main() {
	println("hello")
}
`)

	tree1, err := parser.Parse(domain.Language("go"), content1)
	if err != nil {
		t.Fatal(err)
	}

	tree2, err := parser.Parse(domain.Language("go"), content2)
	if err != nil {
		t.Fatal(err)
	}

	hash1, _ := parser.StructuralHash(tree1)
	hash2, _ := parser.StructuralHash(tree2)

	if hash1 != hash2 {
		t.Fatalf("hashes should match despite comments: %s != %s", hash1, hash2)
	}

	content3 := []byte(`
package main

func main() {
	println("world")
}
`)
	tree3, _ := parser.Parse(domain.Language("go"), content3)
	hash3, _ := parser.StructuralHash(tree3)
	
	if hash1 == hash3 {
		t.Fatalf("hashes should differ for different code")
	}
}
