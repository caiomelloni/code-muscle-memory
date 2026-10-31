package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func initRepo(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	if _, err := git(dir, "init", "-q"); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commitCount(t *testing.T, dir string) string {
	t.Helper()
	n, err := git(dir, "rev-list", "--all", "--count")
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCommitRecordsChange(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeFile(t, filepath.Join(dir, "progress.json"), "{}")

	if err := Commit(dir, "cmm next"); err != nil {
		t.Fatal(err)
	}

	subject, err := git(dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatal(err)
	}
	if subject != "cmm next" {
		t.Fatalf("subject = %q", subject)
	}
	files, err := git(dir, "ls-files")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files, "progress.json") {
		t.Fatalf("progress.json not committed: %q", files)
	}
}

func TestCommitWithoutChangeCreatesNoCommit(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeFile(t, filepath.Join(dir, "progress.json"), "{}")
	if err := Commit(dir, "first"); err != nil {
		t.Fatal(err)
	}

	if err := Commit(dir, "second"); err != nil {
		t.Fatal(err)
	}

	if got := commitCount(t, dir); got != "1" {
		t.Fatalf("commits = %s, want 1", got)
	}
}

func TestCommitOutsideRepoFails(t *testing.T) {
	if err := Commit(t.TempDir(), "cmm next"); err == nil {
		t.Fatal("Commit succeeded outside a git repository")
	}
}

func TestCommitRefusesDirectoryNestedInRepo(t *testing.T) {
	parent := t.TempDir()
	initRepo(t, parent)
	writeFile(t, filepath.Join(parent, "keep.txt"), "x")
	if err := Commit(parent, "first"); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(parent, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(nested, "progress.json"), "{}")

	if err := Commit(nested, "cmm next"); err == nil {
		t.Fatal("Commit succeeded in a directory nested inside a repository")
	}

	if got := commitCount(t, parent); got != "1" {
		t.Fatalf("parent commits = %s, want 1", got)
	}
	staged, err := git(parent, "diff", "--cached", "--name-only")
	if err != nil {
		t.Fatal(err)
	}
	if staged != "" {
		t.Fatalf("parent has staged files: %q", staged)
	}
}
