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
	if err := checkRoot(dir); err != nil {
		return err
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

// ErrNoMessage is returned by Sync when the commit message comes back empty.
var ErrNoMessage = errors.New("no commit message given, so nothing was committed or pushed")

// Sync commits any pending change in dir and pushes it to the configured
// upstream. askMessage supplies the commit message and is only called when
// there is something to commit. Sync never forces a push, so a remote with
// other work is left intact and the error is returned instead.
func Sync(dir string, askMessage func() (string, error)) error {
	if err := checkRoot(dir); err != nil {
		return err
	}
	changes, err := git(dir, "status", "--porcelain")
	if err != nil {
		return err
	}
	if changes != "" {
		message, err := askMessage()
		if err != nil {
			return err
		}
		if message == "" {
			return ErrNoMessage
		}
		if err := Commit(dir, message); err != nil {
			return err
		}
	}
	_, err = git(dir, "push", "-q")
	return err
}

// Status is how the memory repo compares with its upstream.
type Status struct {
	Uncommitted bool
	HasUpstream bool
	Ahead       int
	Behind      int
}

// CheckStatus reports dir's sync status. It does not contact the remote, so
// Behind only counts commits seen by the last push or fetch.
func CheckStatus(dir string) (Status, error) {
	if err := checkRoot(dir); err != nil {
		return Status{}, err
	}
	changes, err := git(dir, "status", "--porcelain")
	if err != nil {
		return Status{}, err
	}
	status := Status{Uncommitted: changes != ""}

	if _, err := git(dir, "rev-parse", "--abbrev-ref", "@{upstream}"); err != nil {
		return status, nil
	}
	status.HasUpstream = true
	counts, err := git(dir, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		return Status{}, err
	}
	if _, err := fmt.Sscan(counts, &status.Ahead, &status.Behind); err != nil {
		return Status{}, fmt.Errorf("reading commit counts %q: %w", counts, err)
	}
	return status, nil
}

// String describes the status and, when something is outstanding, what to run.
func (s Status) String() string {
	var parts []string
	if s.Uncommitted {
		parts = append(parts, "uncommitted changes")
	}
	if s.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("%d %s to push", s.Ahead, plural(s.Ahead)))
	}
	if s.Behind > 0 {
		parts = append(parts, fmt.Sprintf("%d %s behind the remote", s.Behind, plural(s.Behind)))
	}
	if !s.HasUpstream {
		parts = append(parts, "no remote branch set up")
		return strings.Join(parts, ", ")
	}
	if len(parts) == 0 {
		return "up to date"
	}
	summary := strings.Join(parts, ", ")
	if s.Behind > 0 {
		return summary + " (pull in the memory repo before \"cmm sync\")"
	}
	return summary + " (run \"cmm sync\")"
}

func plural(n int) string {
	if n == 1 {
		return "commit"
	}
	return "commits"
}

func checkRoot(dir string) error {
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
	return nil
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
