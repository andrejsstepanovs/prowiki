package ast

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

type HeuristicParser struct{}

func NewHeuristicParser() *HeuristicParser {
	return &HeuristicParser{}
}

func (p *HeuristicParser) Parse(lang domain.Language, content []byte) (*domain.Tree, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var structuralLines []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		structuralLines = append(structuralLines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	normalized := strings.Join(structuralLines, "\n")
	hash := sha256.Sum256([]byte(normalized))
	
	return &domain.Tree{
		Hash: hex.EncodeToString(hash[:]),
	}, nil
}

func (p *HeuristicParser) StructuralHash(tree *domain.Tree) (string, error) {
	return tree.Hash, nil
}
