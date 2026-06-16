package ast

import (
	"strings"
	"testing"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
	"pgregory.net/rapid"
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

// Feature: prowiki-gap-analysis, Property 16: AST hash ignores comments (Go/JS/TS)
func TestASTHashIgnoresCommentsGoJSTS(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		codeLines := rapid.SliceOf(rapid.StringMatching(`^[a-zA-Z0-9_{}() ]+$`)).Draw(rt, "codeLines")
		
		var withComments []string
		var withoutComments []string

		for _, line := range codeLines {
			if strings.TrimSpace(line) != "" {
				withoutComments = append(withoutComments, line)
				withComments = append(withComments, line)
			}
			
			if rapid.Bool().Draw(rt, "insertLineComment") {
				withComments = append(withComments, "// " + rapid.StringMatching(`^[a-zA-Z0-9 ]*$`).Draw(rt, "lineComment"))
			}
			if rapid.Bool().Draw(rt, "insertBlockComment") {
				withComments = append(withComments, "/* \n" + rapid.StringMatching(`^[a-zA-Z0-9 ]*$`).Draw(rt, "blockComment") + "\n */")
			}
		}

		parser := NewHeuristicParser()
		for _, lang := range []string{"go", "javascript", "typescript"} {
			treeWith, _ := parser.Parse(domain.Language(lang), []byte(strings.Join(withComments, "\n")))
			treeWithout, _ := parser.Parse(domain.Language(lang), []byte(strings.Join(withoutComments, "\n")))

			if treeWith.Hash != treeWithout.Hash {
				rt.Fatalf("Hash mismatch for %s: with comments %s, without %s", lang, treeWith.Hash, treeWithout.Hash)
			}
		}
	})
}

// Feature: prowiki-gap-analysis, Property 17: AST hash ignores Python docstrings and comments
func TestASTHashIgnoresPythonDocstringsAndComments(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		codeLines := rapid.SliceOf(rapid.StringMatching(`^[a-zA-Z0-9_=() ]+$`)).Draw(rt, "codeLines")
		
		var withComments []string
		var withoutComments []string

		for _, line := range codeLines {
			if strings.TrimSpace(line) != "" {
				withoutComments = append(withoutComments, line)
				withComments = append(withComments, line)
			}
			
			if rapid.Bool().Draw(rt, "insertLineComment") {
				withComments = append(withComments, "# " + rapid.StringMatching(`^[a-zA-Z0-9 ]*$`).Draw(rt, "lineComment"))
			}
			if rapid.Bool().Draw(rt, "insertTripleDouble") {
				withComments = append(withComments, `"""` + rapid.StringMatching(`^[a-zA-Z0-9 \n]*$`).Draw(rt, "doc1") + `"""`)
			}
			if rapid.Bool().Draw(rt, "insertTripleSingle") {
				withComments = append(withComments, `'''` + rapid.StringMatching(`^[a-zA-Z0-9 \n]*$`).Draw(rt, "doc2") + `'''`)
			}
		}

		parser := NewHeuristicParser()
		treeWith, _ := parser.Parse(domain.Language("python"), []byte(strings.Join(withComments, "\n")))
		treeWithout, _ := parser.Parse(domain.Language("python"), []byte(strings.Join(withoutComments, "\n")))

		if treeWith.Hash != treeWithout.Hash {
			rt.Fatalf("Hash mismatch for Python: with comments %s, without %s", treeWith.Hash, treeWithout.Hash)
		}
	})
}

// Feature: prowiki-gap-analysis, Property 18: AST hash blank-line invariant
func TestASTHashBlankLineInvariant(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		codeLines := rapid.SliceOf(rapid.StringMatching(`^[a-zA-Z0-9_{}()= ]+$`)).Draw(rt, "codeLines")
		
		var withBlanks []string
		var withoutBlanks []string

		for _, line := range codeLines {
			if strings.TrimSpace(line) != "" {
				withoutBlanks = append(withoutBlanks, line)
				withBlanks = append(withBlanks, line)
			}
			
			numBlanks := rapid.IntRange(1, 3).Draw(rt, "numBlanks")
			for i := 0; i < numBlanks; i++ {
				if rapid.Bool().Draw(rt, "isWhitespaceOnly") {
					withBlanks = append(withBlanks, "   \t  ")
				} else {
					withBlanks = append(withBlanks, "")
				}
			}
		}

		parser := NewHeuristicParser()
		langs := []string{"go", "javascript", "typescript", "python", "", "unknown"}
		
		for _, lang := range langs {
			treeWith, _ := parser.Parse(domain.Language(lang), []byte(strings.Join(withBlanks, "\n")))
			treeWithout, _ := parser.Parse(domain.Language(lang), []byte(strings.Join(withoutBlanks, "\n")))

			if treeWith.Hash != treeWithout.Hash {
				rt.Fatalf("Hash mismatch for %s: with blanks %s, without %s", lang, treeWith.Hash, treeWithout.Hash)
			}
		}
	})
}
