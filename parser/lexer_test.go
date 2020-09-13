package parser

import (
	"strings"
	"testing"
)

func TestLexer_Run(t *testing.T) {
	dummyPos := pos(0, 0)

	tests := []struct {
		caption       string
		src           string
		checkPosition bool
		tokens        []*token
	}{
		{
			caption: "the lexer can recognize all kinds of tokens",
			src:     `|:;?*+id_1"pattern"!!! `,
			tokens: []*token{
				newSymbolToken(dummyPos, tokenKindVBar),
				newSymbolToken(dummyPos, tokenKindColon),
				newSymbolToken(dummyPos, tokenKindSemicolon),
				newSymbolToken(dummyPos, tokenKindOptional),
				newSymbolToken(dummyPos, tokenKindZeorOrMore),
				newSymbolToken(dummyPos, tokenKindOneOrMore),
				newIDToken(dummyPos, "id_1"),
				newPatternToken(dummyPos, "pattern"),
				newUnknownToken(dummyPos, "!!!"),
				newEOFToken(dummyPos),
			},
		},
		{
			caption: "the lexer can recognize comments",
			src:     "// This is newline-terminated comment.\n// This is eof-terminated comment.",
			tokens: []*token{
				newCommentToken(dummyPos, " This is newline-terminated comment."),
				newCommentToken(dummyPos, " This is eof-terminated comment."),
			},
		},
		{
			caption: "the lexer can recognize correct format tokens following unknown tokens",
			src:     `!|!:!;!?!*!+!id!"pattern"!/foo/`,
			tokens: []*token{
				newUnknownToken(dummyPos, "!"),
				newSymbolToken(dummyPos, tokenKindVBar),
				newUnknownToken(dummyPos, "!"),
				newSymbolToken(dummyPos, tokenKindColon),
				newUnknownToken(dummyPos, "!"),
				newSymbolToken(dummyPos, tokenKindSemicolon),
				newUnknownToken(dummyPos, "!"),
				newSymbolToken(dummyPos, tokenKindOptional),
				newUnknownToken(dummyPos, "!"),
				newSymbolToken(dummyPos, tokenKindZeorOrMore),
				newUnknownToken(dummyPos, "!"),
				newSymbolToken(dummyPos, tokenKindOneOrMore),
				newUnknownToken(dummyPos, "!"),
				newIDToken(dummyPos, "id"),
				newUnknownToken(dummyPos, "!"),
				newPatternToken(dummyPos, "pattern"),
				newUnknownToken(dummyPos, "!"),
				newUnknownToken(dummyPos, "/"),
				newIDToken(dummyPos, "foo"),
				newUnknownToken(dummyPos, "/"),
				newEOFToken(dummyPos),
			},
		},
		{
			caption:       "the lexer can recognize each position of tokens",
			src:           "a: b;\nc: d;\n",
			checkPosition: true,
			tokens: []*token{
				newIDToken(pos(1, 1), "a"),
				newSymbolToken(pos(1, 2), tokenKindColon),
				newIDToken(pos(1, 4), "b"),
				newSymbolToken(pos(1, 5), tokenKindSemicolon),
				newIDToken(pos(2, 1), "c"),
				newSymbolToken(pos(2, 2), tokenKindColon),
				newIDToken(pos(2, 4), "d"),
				newSymbolToken(pos(2, 5), tokenKindSemicolon),
				newEOFToken(pos(3, 1)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			l := newLexer(strings.NewReader(tt.src))
			for _, eTok := range tt.tokens {
				aTok, err := l.next()
				if err != nil {
					t.Error(err)
					continue
				}
				if !matchToken(eTok, aTok, tt.checkPosition) {
					t.Fatalf("unexpected token; want: %v, got: %v", eTok, aTok)
				}
			}
		})
	}
}

func pos(line, column int) Position {
	return Position{
		Line:   line,
		Column: column,
	}
}

func matchToken(expected, actual *token, checkPosition bool) bool {
	if checkPosition {
		if actual.pos.Line != expected.pos.Line || actual.pos.Column != expected.pos.Column {
			return false
		}
	}
	if actual.kind != expected.kind || actual.text != expected.text {
		return false
	}

	return true
}
