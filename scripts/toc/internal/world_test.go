package internal_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/powerman/check"

	"github.com/powerman/go-scripts-example/scripts/toc/internal"
)

// TestRealWorld checks that commands work on each OS.
//
//nolint:paralleltest // TOCFiles runs git in the current directory, changed by t.Chdir.
func TestRealWorld(tt *testing.T) {
	t := check.T(tt).MustAll()
	dir := t.TempDir()
	t.Chdir(dir)
	out, err := exec.CommandContext(t.Context(), "git", "init", "-q").CombinedOutput()
	t.Nil(err, string(out))

	var w internal.RealWorld

	files, err := w.TOCFiles()
	t.Nil(err)
	t.Len(files, 0)

	t.Nil(os.MkdirAll(filepath.Join("sub dir", "ünï"), 0o750))
	t.Nil(os.WriteFile(".gitignore", []byte("ignored.md\n"), 0o600))
	t.Nil(os.WriteFile("ignored.md", []byte("<!-- toc -->\n"), 0o600))
	t.Nil(os.WriteFile("a.txt", []byte("<!-- toc -->\n"), 0o600))
	t.Nil(os.WriteFile("b.md", []byte("x <!-- toc -->\n"), 0o600))
	t.Nil(os.WriteFile(filepath.Join("sub dir", "ünï", "c d.md"), []byte("x\n<!-- toc -->\n"), 0o600))

	files, err = w.TOCFiles()
	t.Nil(err)
	t.DeepEqual(files, []string{"sub dir/ünï/c d.md"})
}
