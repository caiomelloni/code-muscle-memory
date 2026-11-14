package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"code-muscle-memory/internal/exercise"
)

// shellTimeout bounds every script an exercise runs. A command that waits for
// input it will never get, or loops forever, must not hang the session.
const shellTimeout = 15 * time.Second

type ShellExecutor struct {
	// timeout bounds each script; zero means shellTimeout.
	timeout time.Duration
}

func NewShellExecutor() ShellExecutor {
	return ShellExecutor{}
}

func (s ShellExecutor) limit() time.Duration {
	if s.timeout <= 0 {
		return shellTimeout
	}
	return s.timeout
}

// Evaluate runs the user's command line in a sandbox directory seeded by the
// exercise's setup script, then hands the outcome to the exercise's hidden
// check script, which decides whether the command did what was asked.
//
// The command's own exit status does not decide anything: plenty of useful
// commands report failure by design (a grep that matches nothing), and plenty
// of wrong commands succeed. It is used only to explain a failing check.
func (s ShellExecutor) Evaluate(ctx context.Context, ex exercise.Exercise, submission string) (Result, error) {
	command := stripCommentLines(submission)
	if strings.TrimSpace(command) == "" {
		return Result{Status: CommandError, Output: "You did not write a command. Add one below the instructions and save the file."}, nil
	}

	box, cleanup, err := s.newSandbox()
	if err != nil {
		return Result{}, err
	}
	defer cleanup()

	if err := box.runSetup(ctx, ex); err != nil {
		return Result{}, err
	}

	run, err := box.runCommand(ctx, submission, command)
	if err != nil {
		return Result{}, err
	}
	if run.timedOut {
		return Result{Status: CommandError, Output: fmt.Sprintf("Your command was still running after %s and was stopped. It is probably waiting for input or looping.\n\n%s", s.limit(), indentBlock(command))}, nil
	}

	check, err := box.runCheck(ctx, ex, run)
	if err != nil {
		return Result{}, err
	}
	if check.timedOut {
		return Result{}, fmt.Errorf("exercise %q: check script timed out after %s", ex.ID, s.limit())
	}

	if check.exitCode != 0 {
		return Result{Status: checkFailureStatus(run), Output: failureReport(command, run, check)}, nil
	}

	if missing, ok := missingRequiredCommand(ex.RequiredCommands, command); ok {
		return Result{Status: MissingCommand, Output: fmt.Sprintf("Your command produced the right result, but it does not use %s, which is what this exercise is practising. Solve it with %s.", strconv.Quote(missing), strconv.Quote(missing))}, nil
	}

	return Result{Status: Success, Output: strings.TrimRight(run.stdout, "\n")}, nil
}

// Preview seeds a throwaway sandbox and describes what the user will find in
// it. Shell exercises are answered blind — there is no chance to look around
// before writing the command — so the starting directory has to be shown.
// It is rebuilt from scratch for the graded run, so setup must be deterministic.
func (s ShellExecutor) Preview(ctx context.Context, ex exercise.Exercise) (string, error) {
	if strings.TrimSpace(ex.Setup) == "" {
		return "", nil
	}

	box, cleanup, err := s.newSandbox()
	if err != nil {
		return "", err
	}
	defer cleanup()

	if err := box.runSetup(ctx, ex); err != nil {
		return "", err
	}
	return describeTree(box.work)
}

// VerifyAuthoring checks that the exercise grades what it claims to: the setup
// script has to run, the reference solution has to satisfy the check, and a
// command that does nothing must not — a check that passes a no-op verifies
// nothing, the shell equivalent of hidden tests a blank stub already passes.
func (s ShellExecutor) VerifyAuthoring(ctx context.Context, ex exercise.Exercise) ([]string, error) {
	var problems []string

	if strings.TrimSpace(ex.Setup) != "" {
		box, cleanup, err := s.newSandbox()
		if err != nil {
			return nil, err
		}
		err = box.runSetup(ctx, ex)
		cleanup()
		if err != nil {
			problems = append(problems, err.Error())
			return problems, nil
		}
	}

	if strings.TrimSpace(ex.Solution) == "" {
		problems = append(problems, "has no solution, so the check script cannot be verified; add one")
	} else {
		result, err := s.Evaluate(ctx, ex, ex.Solution)
		if err != nil {
			return nil, err
		}
		if result.Status != Success {
			problems = append(problems, "solution does not satisfy the check script:\n"+result.Output)
		}
	}

	result, err := s.Evaluate(ctx, ex, ":")
	if err != nil {
		return nil, err
	}
	if result.Status == Success {
		problems = append(problems, "a command that does nothing already satisfies the check script, so it verifies nothing")
	}
	return problems, nil
}

// sandbox is the directory tree an exercise runs in: work is the seeded
// directory the scripts see as their working directory, and io holds the
// captured output, kept outside work so that listing or globbing the working
// directory does not turn up the grader's own files.
type sandbox struct {
	work    string
	io      string
	timeout time.Duration
}

func (s ShellExecutor) newSandbox() (*sandbox, func(), error) {
	root, err := os.MkdirTemp("", "cmm-shell-*")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(root) }

	box := &sandbox{work: filepath.Join(root, "work"), io: filepath.Join(root, "io"), timeout: s.limit()}
	for _, dir := range []string{box.work, box.io} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			cleanup()
			return nil, nil, err
		}
	}
	return box, cleanup, nil
}

func (b *sandbox) runSetup(ctx context.Context, ex exercise.Exercise) error {
	if strings.TrimSpace(ex.Setup) == "" {
		return nil
	}
	run, err := b.run(ctx, abortOnError(ex.Setup), nil)
	if err != nil {
		return err
	}
	if run.timedOut {
		return fmt.Errorf("exercise %q: setup script timed out after %s", ex.ID, b.timeout)
	}
	if run.exitCode != 0 {
		return fmt.Errorf("exercise %q: setup script failed with status %d:\n%s", ex.ID, run.exitCode, strings.TrimRight(run.stdout+run.stderr, "\n"))
	}
	return nil
}

// runCommand executes the file exactly as the user saved it — the shell knows
// how to ignore the instruction comments, and stripping them by hand could
// mangle a here-document. The stripped text is carried alongside for reporting
// and for the check script, which must see the command, not the instructions.
func (b *sandbox) runCommand(ctx context.Context, submission, command string) (runResult, error) {
	run, err := b.run(ctx, submission, nil)
	run.script = command
	return run, err
}

// runCheck runs the hidden check script with the command's captured output
// reachable through CMM_ environment variables, so a check can assert on what
// the command printed, what it left behind in the directory, or both.
func (b *sandbox) runCheck(ctx context.Context, ex exercise.Exercise, run runResult) (runResult, error) {
	files := map[string]string{
		"stdout":  run.stdout,
		"stderr":  run.stderr,
		"command": run.script,
	}
	env := []string{"CMM_EXIT=" + strconv.Itoa(run.exitCode)}
	for name, content := range files {
		path := filepath.Join(b.io, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return runResult{}, err
		}
		env = append(env, "CMM_"+strings.ToUpper(name)+"="+path)
	}
	sort.Strings(env)
	return b.run(ctx, abortOnError(ex.Check), env)
}

// abortOnError makes an exercise's own script stop at its first failing line,
// so a check written as a list of assertions fails on the one that broke
// instead of on whatever happened to run last.
func abortOnError(script string) string {
	return "set -e\n" + script
}

type runResult struct {
	script   string
	stdout   string
	stderr   string
	exitCode int
	timedOut bool
}

func (b *sandbox) run(ctx context.Context, script string, extraEnv []string) (runResult, error) {
	ctx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, shellPath(), "-c", script)
	cmd.Dir = b.work
	// HOME points into the sandbox so an exercise about "~" cannot reach the
	// real home directory, and LC_ALL fixes collation so sorted output is the
	// same on every machine the deck runs on.
	cmd.Env = append(os.Environ(), "HOME="+b.work, "PWD="+b.work, "LC_ALL=C")
	cmd.Env = append(cmd.Env, extraEnv...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	result := runResult{script: script, stdout: stdout.String(), stderr: stderr.String()}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.timedOut = true
		return result, nil
	}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		result.exitCode = exitErr.ExitCode()
	default:
		return runResult{}, err
	}
	return result, nil
}

// shellPath prefers bash, which is what a "learn the command line" deck is
// written against, and falls back to whatever /bin/sh is on systems without it.
func shellPath() string {
	if path, err := exec.LookPath("bash"); err == nil {
		return path
	}
	return "/bin/sh"
}

// checkFailureStatus distinguishes a command that ran fine but did the wrong
// thing from one that never ran properly at all — a typo, a missing utility, a
// bad flag — because the two need different things pointed out to the user.
func checkFailureStatus(run runResult) Status {
	if run.exitCode != 0 {
		return CommandError
	}
	return CheckFailure
}

// failureReport leads with whatever the check script said, since that is the
// exercise explaining in its own words what it wanted, and follows it with the
// command and what came of it as the evidence.
func failureReport(command string, run runResult, check runResult) string {
	var b strings.Builder
	message := strings.TrimSpace(check.stdout + check.stderr)
	if message == "" {
		message = "Your command did not do what the exercise asked for."
	}
	fmt.Fprintf(&b, "%s\n\nYour command:\n%s", message, indentBlock(command))

	if run.exitCode != 0 {
		fmt.Fprintf(&b, "It exited with status %d.\n", run.exitCode)
	}
	appendSection(&b, "It printed", run.stdout)
	appendSection(&b, "It reported", run.stderr)
	return strings.TrimRight(b.String(), "\n")
}

func appendSection(b *strings.Builder, label, body string) {
	if strings.TrimSpace(body) == "" {
		return
	}
	b.WriteString(label + ":\n")
	b.WriteString(indentBlock(body))
}

func indentBlock(text string) string {
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		b.WriteString("    " + line + "\n")
	}
	return b.String()
}

// stripCommentLines drops the instruction header the editable file starts with,
// leaving what the user actually typed. Only whole comment lines go: a "#"
// inside a command is an argument, not a comment.
func stripCommentLines(submission string) string {
	var kept []string
	for _, line := range strings.Split(submission, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

var commandWord = regexp.MustCompile(`[A-Za-z0-9_./-]+`)

// missingRequiredCommand reports the first utility the exercise requires that
// the user's command line does not mention. Matching is by word, so a utility
// named by its full path still counts.
func missingRequiredCommand(required []string, command string) (string, bool) {
	if len(required) == 0 {
		return "", false
	}

	used := map[string]struct{}{}
	for _, word := range commandWord.FindAllString(command, -1) {
		used[word] = struct{}{}
		used[filepath.Base(word)] = struct{}{}
	}
	for _, name := range required {
		if _, ok := used[name]; !ok {
			return name, true
		}
	}
	return "", false
}

const (
	previewMaxFileSize  = 400
	previewMaxFileLines = 12
)

// describeTree lists the seeded directory and shows the contents of the small
// text files in it, which is the information a user would otherwise get by
// looking around before answering.
func describeTree(root string) (string, error) {
	var entries []string
	var contents []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			return err
		}
		if d.IsDir() {
			entries = append(entries, rel+"/")
			return nil
		}
		entries = append(entries, rel)
		if body, ok := previewableFile(path, d); ok {
			contents = append(contents, rel+":\n"+indentBlock(body))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "The working directory starts out empty.", nil
	}

	sort.Strings(entries)
	report := "The working directory contains:\n" + indentBlock(strings.Join(entries, "\n"))
	if len(contents) > 0 {
		sort.Strings(contents)
		report += "\n" + strings.Join(contents, "\n")
	}
	return strings.TrimRight(report, "\n"), nil
}

func previewableFile(path string, d os.DirEntry) (string, bool) {
	info, err := d.Info()
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > previewMaxFileSize {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil || !utf8.Valid(data) || bytes.ContainsRune(data, 0) {
		return "", false
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > previewMaxFileLines {
		lines = append(lines[:previewMaxFileLines], "...")
	}
	return strings.Join(lines, "\n"), true
}
