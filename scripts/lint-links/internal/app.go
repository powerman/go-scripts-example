package internal

import (
	"fmt"
	"io"
	urlpkg "net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/powerman/go-scripts-example/scripts/internal/markdown"
)

type app struct {
	world  World
	stdout io.Writer
	docs   map[string]*markdown.Doc
	paths  map[string]bool // Files and directories of the repository.
}

// NewApp creates an App which uses world and prints broken links to stdout.
func NewApp(world World, stdout io.Writer) *app {
	return &app{
		world:  world,
		stdout: stdout,
		docs:   make(map[string]*markdown.Doc),
		paths:  make(map[string]bool),
	}
}

// Check implements [App].
func (a *app) Check(files []string) error {
	all, err := a.world.GitLsFiles()
	if err != nil {
		return fmt.Errorf("git ls-files: %w", err)
	}
	for _, name := range all {
		for p := name; p != "."; p = path.Dir(p) {
			a.paths[p] = true
		}
	}

	if len(files) == 0 {
		for _, name := range all {
			if path.Ext(name) == ".md" {
				files = append(files, name)
			}
		}
	}

	broken := 0
	for _, name := range files {
		// Paths of the repository use '/', while arguments may use '\' on Windows.
		n, err := a.checkFile(path.Clean(filepath.ToSlash(name)))
		if err != nil {
			return err
		}
		broken += n
	}

	if broken > 0 {
		return fmt.Errorf("%w: %d", ErrBrokenLinks, broken)
	}
	return nil
}

// checkFile prints broken links of the file and returns their amount.
func (a *app) checkFile(name string) (broken int, _ error) {
	doc, err := a.parse(name)
	if err != nil {
		return 0, err
	}
	for _, link := range doc.Links {
		problem, err := a.checkLink(name, link.Dest)
		if err != nil {
			return 0, err
		}
		if problem != "" {
			broken++
			_, err = fmt.Fprintf(a.stdout, "%s:%d: %s\n", name, link.Line, problem)
			if err != nil {
				return 0, err
			}
		}
	}
	return broken, nil
}

func (a *app) parse(name string) (*markdown.Doc, error) {
	doc, ok := a.docs[name]
	if !ok {
		src, err := a.world.ReadFile(name)
		if err != nil {
			return nil, err
		}
		doc = markdown.Parse(src)
		a.docs[name] = doc
	}
	return doc, nil
}

// checkLink returns a description of a problem with the link dest in the file from,
// or an empty string if the link is fine.
func (a *app) checkLink(from, dest string) (string, error) {
	u, err := urlpkg.Parse(dest)
	switch {
	case err != nil:
		return fmt.Sprintf("invalid link %q", dest), nil //nolint:nilerr // A problem of the file.
	case u.Scheme != "" || u.Host != "":
		return "", nil // External links are out of scope.
	}

	target, fragment := u.Path, u.Fragment
	switch {
	case target == "":
		target = from
	case strings.HasPrefix(target, "/"): // GitHub resolves it from the root of the repository.
		target = path.Clean(target[1:])
	default:
		target = path.Join(path.Dir(from), target)
	}
	if !a.paths[target] {
		return fmt.Sprintf("link %q: no %s in the repository", dest, target), nil
	}

	if fragment == "" || path.Ext(target) != ".md" {
		return "", nil
	}
	doc, err := a.parse(target)
	if err != nil {
		return "", err
	}
	_, ok := doc.HeadingByID(fragment)
	if !ok {
		return fmt.Sprintf("link %q: no heading #%s in %s", dest, fragment, target), nil
	}
	return "", nil
}
