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

// initRepoWithRemote returns a repository whose main branch tracks a new bare
// remote, along with that remote's path.
func initRepoWithRemote(t *testing.T) (dir, remote string) {
	t.Helper()
	dir, remote = t.TempDir(), t.TempDir()
	initRepo(t, dir)
	if _, err := git(remote, "init", "-q", "--bare", "-b", "main"); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "progress.json"), "{}")
	if err := Commit(dir, "first"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"branch", "-M", "main"},
		{"remote", "add", "origin", remote},
		{"push", "-q", "-u", "origin", "main"},
	} {
		if _, err := git(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	return dir, remote
}

func TestSyncCommitsAndPushes(t *testing.T) {
	dir, remote := initRepoWithRemote(t)
	writeFile(t, filepath.Join(dir, "progress.json"), `{"a":1}`)

	if err := Sync(dir, "cmm sync"); err != nil {
		t.Fatal(err)
	}

	subject, err := git(remote, "log", "-1", "--format=%s", "main")
	if err != nil {
		t.Fatal(err)
	}
	if subject != "cmm sync" {
		t.Fatalf("remote subject = %q", subject)
	}
}

func TestSyncPushesEarlierCommitsWithNoPendingChange(t *testing.T) {
	dir, remote := initRepoWithRemote(t)
	writeFile(t, filepath.Join(dir, "progress.json"), `{"a":1}`)
	if err := Commit(dir, "cmm next"); err != nil {
		t.Fatal(err)
	}

	if err := Sync(dir, "cmm sync"); err != nil {
		t.Fatal(err)
	}

	subject, err := git(remote, "log", "-1", "--format=%s", "main")
	if err != nil {
		t.Fatal(err)
	}
	if subject != "cmm next" {
		t.Fatalf("remote subject = %q", subject)
	}
}

func TestSyncKeepsCommitWhenPushFails(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeFile(t, filepath.Join(dir, "progress.json"), "{}")

	if err := Sync(dir, "cmm sync"); err == nil {
		t.Fatal("Sync succeeded without a remote")
	}

	if got := commitCount(t, dir); got != "1" {
		t.Fatalf("commits = %s, want 1", got)
	}
}

func TestSyncDoesNotForceOverDivergedRemote(t *testing.T) {
	dir, remote := initRepoWithRemote(t)

	other := t.TempDir()
	if _, err := git(other, "clone", "-q", remote, "."); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(other, "other.txt"), "x")
	if err := Commit(other, "from another machine"); err != nil {
		t.Fatal(err)
	}
	if _, err := git(other, "push", "-q"); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(dir, "progress.json"), `{"a":1}`)
	if err := Sync(dir, "cmm sync"); err == nil {
		t.Fatal("Sync succeeded against a diverged remote")
	}

	subject, err := git(remote, "log", "-1", "--format=%s", "main")
	if err != nil {
		t.Fatal(err)
	}
	if subject != "from another machine" {
		t.Fatalf("remote was overwritten: subject = %q", subject)
	}
}
