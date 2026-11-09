// Package markdown extracts headings, links and HTML comments from a markdown document,
// with heading anchors generated the same way as GitHub does.
package markdown

import (
	"bytes"
	"strconv"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// Heading of a document.
type Heading struct {
	Level int
	// Label is the content of the heading as inline markdown which is safe to use as a link text:
	// it keeps code spans and drops links and emphasis.
	Label string
	// ID is the anchor of the heading without '#'.
	ID string
	// Offset is a byte offset of the line with the heading.
	Offset int
}

// Link (or image) of a document.
type Link struct {
	// Dest is the destination as it is written in the document.
	Dest string
	Line int
}

// Comment is an HTML comment which is a whole block of a document.
type Comment struct {
	// Text is the content of the comment without delimiters and surrounding spaces.
	Text string
	// Start is a byte offset of the first line of the block.
	Start int
	// End is a byte offset right after the last line of the block (including line end).
	End int
}

// Doc is a parsed markdown document.
type Doc struct {
	Headings []Heading
	Links    []Link
	Comments []Comment
}

// Parse parses a markdown document in the GitHub Flavored Markdown dialect.
func Parse(src []byte) *Doc {
	p := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser()
	root := p.Parse(text.NewReader(src))

	doc := &Doc{}
	slugs := newSlugger()
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Heading:
			plain, label := inlineText(n, src)
			doc.Headings = append(doc.Headings, Heading{
				Level:  n.Level,
				Label:  label,
				ID:     slugs.slug(plain),
				Offset: lineStart(src, offsetOf(n)),
			})
		case *ast.Link:
			doc.Links = append(doc.Links, Link{Dest: string(n.Destination), Line: lineOf(src, offsetOf(n))})
		case *ast.Image:
			doc.Links = append(doc.Links, Link{Dest: string(n.Destination), Line: lineOf(src, offsetOf(n))})
		case *ast.HTMLBlock:
			if c, ok := comment(n, src); ok {
				doc.Comments = append(doc.Comments, c)
			}
		}
		return ast.WalkContinue, nil
	})
	return doc
}

// Comment returns the first comment with given content.
func (d *Doc) Comment(content string) (Comment, bool) {
	for _, c := range d.Comments {
		if c.Text == content {
			return c, true
		}
	}
	return Comment{}, false
}

// HeadingByID returns the heading with given anchor.
func (d *Doc) HeadingByID(id string) (Heading, bool) {
	for _, h := range d.Headings {
		if h.ID == id {
			return h, true
		}
	}
	return Heading{}, false
}

// inlineText returns the text of inline content of n
// both as a plain text (used for anchors) and as a link label.
func inlineText(n ast.Node, src []byte) (plain, label string) {
	var p, l strings.Builder
	var walk func(n ast.Node)
	walk = func(n ast.Node) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch c := c.(type) {
			case *ast.Text:
				s := string(c.Segment.Value(src))
				if c.SoftLineBreak() || c.HardLineBreak() {
					s += " "
				}
				p.WriteString(s)
				l.WriteString(escapeLabel(s))
			case *ast.String:
				p.Write(c.Value)
				l.WriteString(escapeLabel(string(c.Value)))
			case *ast.CodeSpan:
				var code strings.Builder
				for t := c.FirstChild(); t != nil; t = t.NextSibling() {
					if t, ok := t.(*ast.Text); ok {
						code.Write(t.Segment.Value(src))
					}
				}
				p.WriteString(code.String())
				l.WriteString(codeSpan(code.String()))
			case *ast.AutoLink:
				p.Write(c.Label(src))
				l.WriteString(escapeLabel(string(c.Label(src))))
			case *ast.RawHTML:
			default:
				walk(c)
			}
		}
	}
	walk(n)
	return strings.TrimSpace(p.String()), strings.TrimSpace(l.String())
}

// escapeLabel escapes brackets which would break a link label.
// Text is taken from markdown source, so other special characters are either escaped already
// or don't start an inline element.
func escapeLabel(s string) string {
	var b strings.Builder
	escaped := false
	for _, r := range s {
		if !escaped && (r == '[' || r == ']') {
			b.WriteByte('\\')
		}
		escaped = !escaped && r == '\\'
		b.WriteRune(r)
	}
	return b.String()
}

// codeSpan returns s as a code span, using enough backticks to keep backticks inside s.
func codeSpan(s string) string {
	fence := "`"
	for strings.Contains(s, fence) {
		fence += "`"
	}
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		s = " " + s + " "
	}
	return fence + s + fence
}

// slugger generates unique anchors like github-slugger used by GitHub.
type slugger struct {
	occurrences map[string]int
}

func newSlugger() *slugger { return &slugger{occurrences: make(map[string]int)} }

func (s *slugger) slug(heading string) string {
	base := slugify(heading)
	slug := base
	for {
		_, taken := s.occurrences[slug]
		if !taken {
			break
		}
		s.occurrences[base]++
		slug = base + "-" + strconv.Itoa(s.occurrences[base])
	}
	s.occurrences[slug] = 0
	return slug
}

// slugify approximates github-slugger:
// it lowercases the text, removes punctuation and symbols, and replaces each space by a hyphen.
func slugify(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == ' ':
			return '-'
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r):
			return unicode.ToLower(r)
		default:
			return -1
		}
	}, s)
}

func comment(n *ast.HTMLBlock, src []byte) (Comment, bool) {
	if n.HTMLBlockType != ast.HTMLBlockType2 || n.Lines().Len() == 0 {
		return Comment{}, false
	}
	lines := n.Lines()
	s := string(lines.Value(src))
	last := lines.At(lines.Len() - 1)
	if n.HasClosure() {
		last = n.ClosureLine
		s += string(last.Value(src))
	}
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "<!--") || !strings.HasSuffix(s, "-->") {
		return Comment{}, false
	}
	return Comment{
		Text:  strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "<!--"), "-->")),
		Start: lineStart(src, lines.At(0).Start),
		End:   lineEnd(src, last.Stop-1),
	}, true
}

// offsetOf returns a byte offset of the source of n.
// Goldmark sets positions of all nodes of a parsed document, so -1 isn't expected.
func offsetOf(n ast.Node) int {
	return max(n.Pos(), 0)
}

func lineOf(src []byte, offset int) int {
	return 1 + bytes.Count(src[:offset], []byte("\n"))
}

func lineStart(src []byte, offset int) int {
	return bytes.LastIndexByte(src[:offset], '\n') + 1
}

func lineEnd(src []byte, offset int) int {
	offset = max(offset, 0)
	i := bytes.IndexByte(src[offset:], '\n')
	if i == -1 {
		return len(src)
	}
	return offset + i + 1
}
