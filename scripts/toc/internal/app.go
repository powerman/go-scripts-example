package internal

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/powerman/go-scripts-example/scripts/internal/markdown"
)

const maxLevel = 6

type app struct {
	world  World
	stdout io.Writer
}

// NewApp creates an App which uses world and prints names of updated files to stdout.
func NewApp(world World, stdout io.Writer) *app {
	return &app{world: world, stdout: stdout}
}

// Update implements [App].
func (a *app) Update(opts Options) error {
	if opts.MinLevel < 1 || opts.MaxLevel > maxLevel || opts.MinLevel > opts.MaxLevel {
		return fmt.Errorf("%w: %d..%d", ErrInvalidLevels, opts.MinLevel, opts.MaxLevel)
	}

	files := opts.Files
	if len(files) == 0 {
		var err error
		files, err = a.world.TOCFiles()
		if err != nil {
			return fmt.Errorf("find files: %w", err)
		}
	}

	var errs []error
	for _, name := range files {
		err := a.updateFile(name, opts)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

func (a *app) updateFile(name string, opts Options) error {
	src, err := a.world.ReadFile(name)
	if err != nil {
		return err
	}

	doc := markdown.Parse(src)
	start, ok := doc.Comment(MarkerStart)
	switch {
	case !ok && len(opts.Files) == 0:
		return nil // Found by TOCFiles, but the marker isn't a comment (e.g., it's in code).
	case !ok:
		return ErrNoMarker
	}
	end, ok := doc.Comment(MarkerEnd)
	if !ok || end.Start < start.End {
		return ErrNoEndMarker
	}

	toc := render(doc.Headings, end.End, opts)
	updated := slices.Concat(src[:start.End], toc, src[end.Start:])
	switch {
	case bytes.Equal(updated, src):
		return nil
	case opts.Check:
		return ErrOutdated
	}

	err = a.world.WriteFile(name, updated)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.stdout, "updated", name)
	return err
}

// render returns a table of contents with a blank line before and after it.
// It includes headings after the given offset.
func render(headings []markdown.Heading, after int, opts Options) []byte {
	var buf bytes.Buffer
	buf.WriteString("\n")
	depth := -1
	for _, h := range headings {
		if h.Offset < after || h.Level < opts.MinLevel || h.Level > opts.MaxLevel {
			continue
		}
		// A list item can't be nested deeper than one level below the previous one.
		depth = min(h.Level-opts.MinLevel, depth+1)
		fmt.Fprintf(&buf, "%s- [%s](#%s)\n", strings.Repeat("  ", depth), h.Label, h.ID)
	}
	if depth >= 0 {
		buf.WriteString("\n")
	}
	return buf.Bytes()
}
