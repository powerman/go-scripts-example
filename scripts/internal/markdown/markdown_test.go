package markdown_test

import (
	"strings"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/go-scripts-example/scripts/internal/markdown"
)

func TestParse_Headings(tt *testing.T) {
	tt.Parallel()

	tests := []struct {
		name string
		src  string
		want []markdown.Heading
	}{
		{
			name: "Levels",
			src:  "# Title\n\ntext\n\n## Section\n\n### Sub section\n",
			want: []markdown.Heading{
				{Level: 1, Label: "Title", ID: "title", Offset: 0},
				{Level: 2, Label: "Section", ID: "section", Offset: 15},
				{Level: 3, Label: "Sub section", ID: "sub-section", Offset: 27},
			},
		},
		{
			name: "Setext",
			src:  "Title\n=====\n\nSection\n-------\n",
			want: []markdown.Heading{
				{Level: 1, Label: "Title", ID: "title", Offset: 0},
				{Level: 2, Label: "Section", ID: "section", Offset: 13},
			},
		},
		{
			name: "Punctuation",
			src:  "## What's new? (v1.2, 2026)\n## A -- B_c\n",
			want: []markdown.Heading{
				{Level: 2, Label: "What's new? (v1.2, 2026)", ID: "whats-new-v12-2026", Offset: 0},
				{Level: 2, Label: "A -- B_c", ID: "a----b_c", Offset: 28},
			},
		},
		{
			name: "Inline",
			src:  "## Use `go run` with *mise* and [Kong](https://example.com)\n",
			want: []markdown.Heading{
				{Level: 2, Label: "Use `go run` with mise and Kong", ID: "use-go-run-with-mise-and-kong"},
			},
		},
		{
			name: "CodeWithBackticks",
			src:  "## Run ``a`b``\n",
			want: []markdown.Heading{
				{Level: 2, Label: "Run ``a`b``", ID: "run-ab"},
			},
		},
		{
			name: "Escaping",
			src:  "## \\[x\\] \\*y\\* [z\n",
			want: []markdown.Heading{
				{Level: 2, Label: "\\[x\\] \\*y\\* \\[z", ID: "x-y-z"},
			},
		},
		{
			name: "Unicode",
			src:  "## Привет, Мир!\n",
			want: []markdown.Heading{
				{Level: 2, Label: "Привет, Мир!", ID: "привет-мир"},
			},
		},
		{
			name: "Duplicates",
			src:  "## A\n## A\n## A-1\n## A\n",
			want: []markdown.Heading{
				{Level: 2, Label: "A", ID: "a", Offset: 0},
				{Level: 2, Label: "A", ID: "a-1", Offset: 5},
				{Level: 2, Label: "A-1", ID: "a-1-1", Offset: 10},
				{Level: 2, Label: "A", ID: "a-2", Offset: 17},
			},
		},
		{
			name: "CodeBlock",
			src:  "```sh\n# not a heading\n```\n",
		},
		{
			name: "CRLF",
			src:  "# Title\r\n\r\n## Section\r\n",
			want: []markdown.Heading{
				{Level: 1, Label: "Title", ID: "title", Offset: 0},
				{Level: 2, Label: "Section", ID: "section", Offset: 11},
			},
		},
	}
	for _, tc := range tests {
		tt.Run(tc.name, func(tt *testing.T) {
			tt.Parallel()
			t := check.T(tt).MustAll()
			doc := markdown.Parse([]byte(tc.src))
			t.DeepEqual(doc.Headings, tc.want)
		})
	}
}

func TestParse_Links(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()

	src := strings.Join([]string{
		"# [Title](#title)",                      // 1
		"",                                       // 2
		"Text [inline](a.md) and ![img](i.png).", // 3
		"Wrapped [ref][r]",                       // 4
		"and <https://example.com>.",             // 5
		"",                                       // 6
		"| a | b |",                              // 7
		"| - | - |",                              // 8
		"| [cell](dir/) | [`code`](c.md#x) |",    // 9
		"",                                       // 10
		"```md",                                  // 11
		"[not a link](x.md)",                     // 12
		"```",                                    // 13
		"",                                       // 14
		"[r]: <r e f.md>",                        // 15
		"[](empty.md)",                           // 16
	}, "\r\n")
	doc := markdown.Parse([]byte(src))
	t.DeepEqual(doc.Links, []markdown.Link{
		{Dest: "#title", Line: 1},
		{Dest: "a.md", Line: 3},
		{Dest: "i.png", Line: 3},
		{Dest: "r e f.md", Line: 4},
		{Dest: "dir/", Line: 9},
		{Dest: "c.md#x", Line: 9},
		{Dest: "empty.md", Line: 16},
	})
}

func TestParse_Comments(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()

	src := "text\n\n<!-- one -->\n\n<!--\n  two\n-->\n\n```\n<!-- code -->\n```\n\n<div>\n<!-- html -->\n</div>\n\n<!-- last -->"
	doc := markdown.Parse([]byte(src))
	t.DeepEqual(doc.Comments, []markdown.Comment{
		{Text: "one", Start: 6, End: 19},
		{Text: "two", Start: 20, End: 35},
		{Text: "last", Start: 87, End: 100},
	})

	c, ok := doc.Comment("two")
	t.True(ok)
	t.Equal(src[c.Start:c.End], "<!--\n  two\n-->\n")
	_, ok = doc.Comment("code")
	t.False(ok)
}

func TestDoc_HeadingByID(tt *testing.T) {
	tt.Parallel()
	t := check.T(tt).MustAll()

	doc := markdown.Parse([]byte("# A\n## B\n"))
	h, ok := doc.HeadingByID("b")
	t.True(ok)
	t.Equal(h.Level, 2)
	_, ok = doc.HeadingByID("c")
	t.False(ok)
}
