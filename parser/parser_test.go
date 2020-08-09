package parser

import (
	"errors"
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	tests := []struct {
		caption     string
		src         string
		syntaxError bool
	}{
		{
			caption: "when a source is in the correct format, the parser can recognize it",
			src:     `a: a b c | c; c: c d e | e; d: "(" e ")"; e: "foo";`,
		},
		{
			caption: "when a source is in the correct format (it contains non-empty productions), the parser can recognize it",
			src:     `a: ; b: | ; c: | d | ;`,
		},
		{
			caption:     "when a source contains an unknown token, the parser raises a syntax error",
			src:         `a: ?;`,
			syntaxError: true,
		},
		{
			caption:     "when a source contains a production that lacks the LHS, the parser raises a syntax error",
			src:         `: b;`,
			syntaxError: true,
		},
		{
			caption:     "when a source contains a production that lacks \":\" (delimiter), the parser raises a syntax error",
			src:         `a b;`,
			syntaxError: true,
		},
		{
			caption:     "when a source contains a production that lacks \";\" (terminator), the parser raises a syntax error",
			src:         `a: b`,
			syntaxError: true,
		},
		{
			caption:     "when a source contains a production that lacks the LHS and the RHS, the parser raises a syntax error",
			src:         `;`,
			syntaxError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			parser, err := NewParser(strings.NewReader(tt.src))
			if err != nil {
				t.Fatalf("failed to create a new parser: %v", err)
			}

			ast, err := parser.Parse()
			if tt.syntaxError {
				syntaxErr := &SyntaxError{}
				if errors.Is(err, syntaxErr) {
					t.Fatalf("error type is mismatched; wont: %T, got: %T", syntaxErr, err)
				}
				if ast != nil {
					t.Fatalf("AST is not nil")
				}
			} else {
				if err != nil {
					t.Fatalf("the parser raised an error: %v", err)
				}
				if ast == nil {
					t.Fatalf("AST is nil")
				}
			}
		})
	}
}
