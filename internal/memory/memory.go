// Package memory records changes to the user's exercises and progress in the
// git repository that holds them.
package memory

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Commit stages everything in dir and commits it. Staging everything means a
// change missed by an earlier run goes in with this one. It does nothing when
// there is no change, and it refuses to act unless dir is the root of a git
// repository, so it can never commit into a repository that merely contains dir.
func Commit(dir, message string) error {
	root, err := git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	same, err := samePath(root, dir)
	if err != nil {
		return err
	}
	if !same {
		return fmt.Errorf("%s is not the root of a git repository (root is %s)", dir, root)
	}

	if _, err := git(dir, "add", "-A"); err != nil {
		return err
	}
	staged, err := git(dir, "diff", "--cached", "--name-only")
	if err != nil {
		return err
	}
	if staged == "" {
		return nil
	}
	_, err = git(dir, "commit", "-q", "-m", message)
	return err
}

// Sync commits any pending change in dir and pushes it to the configured
// upstream. It never forces a push, so a remote with other work is left intact
// and the error is returned instead.
func Sync(dir, message string) error {
	if err := Commit(dir, message); err != nil {
		return err
	}
	_, err := git(dir, "push", "-q")
	return err
}

func samePath(a, b string) (bool, error) {
	a, err := filepath.EvalSymlinks(a)
	if err != nil {
		return false, err
	}
	b, err = filepath.EvalSymlinks(b)
	if err != nil {
		return false, err
	}
	return a == b, nil
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New("git " + args[0] + ": " + msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}
