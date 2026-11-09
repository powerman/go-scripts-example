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
//nolint:paralleltest // GitLsFiles runs git in the current directory, changed by t.Chdir.
func TestRealWorld(tt *testing.T) {
	t := check.T(tt).MustAll()
	dir := t.TempDir()
	t.Chdir(dir)
	out, err := exec.CommandContext(t.Context(), "git", "init", "-q").CombinedOutput()
	t.Nil(err, string(out))
	t.Nil(os.MkdirAll(filepath.Join("sub dir", "ünï"), 0o750))
	t.Nil(os.WriteFile(".gitignore", []byte("ignored.md\n"), 0o600))
	t.Nil(os.WriteFile("ignored.md", nil, 0o600))
	t.Nil(os.WriteFile("a.txt", nil, 0o600))
	t.Nil(os.WriteFile(filepath.Join("sub dir", "ünï", "b c.md"), nil, 0o600))

	var w internal.RealWorld

	files, err := w.GitLsFiles("*.md")
	t.Nil(err)
	t.DeepEqual(files, []string{"sub dir/ünï/b c.md"})

	files, err = w.GitLsFiles("*.none")
	t.Nil(err)
	t.Len(files, 0)

	t.Nil(w.WriteFile("sub dir/ünï/b c.md", []byte("data\r\n")))
	data, err := w.ReadFile("sub dir/ünï/b c.md")
	t.Nil(err)
	t.Equal(string(data), "data\r\n")

	_, err = w.ReadFile("none.md")
	t.Err(err, os.ErrNotExist)
}
