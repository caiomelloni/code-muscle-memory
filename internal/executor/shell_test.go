package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"code-muscle-memory/internal/exercise"
)

func countingExercise() exercise.Exercise {
	return exercise.Exercise{
		ID:         "sh-001",
		Language:   exercise.LanguageShell,
		Setup:      "printf 'a\\nb\\nc\\n' > notes.txt\n",
		Check:      "[ \"$(tr -d '[:space:]' < \"$CMM_STDOUT\")\" = 3 ] || { echo 'expected the number 3'; exit 1; }\n",
		Solution:   "wc -l < notes.txt\n",
		Difficulty: 1,
	}
}

func TestShellEvaluateAcceptsACorrectCommand(t *testing.T) {
	result, err := NewShellExecutor().Evaluate(context.Background(), countingExercise(), "wc -l < notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != Success {
		t.Fatalf("Evaluate() = %q, want %q\n%s", result.Status, Success, result.Output)
	}
}

func TestShellEvaluateReportsAWrongResult(t *testing.T) {
	result, err := NewShellExecutor().Evaluate(context.Background(), countingExercise(), "wc -c < notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != CheckFailure {
		t.Fatalf("Evaluate() = %q, want %q", result.Status, CheckFailure)
	}
	if !strings.Contains(result.Output, "expected the number 3") {
		t.Errorf("output does not carry the check's message:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, "wc -c < notes.txt") {
		t.Errorf("output does not show the command that ran:\n%s", result.Output)
	}
}

// A command that never ran properly is reported differently from one that ran
// and did the wrong thing, because a typo needs the shell's complaint shown.
func TestShellEvaluateReportsACommandThatFailedToRun(t *testing.T) {
	result, err := NewShellExecutor().Evaluate(context.Background(), countingExercise(), "wcc -l < notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != CommandError {
		t.Fatalf("Evaluate() = %q, want %q", result.Status, CommandError)
	}
	if !strings.Contains(result.Output, "not found") {
		t.Errorf("output does not include what the shell reported:\n%s", result.Output)
	}
}

func TestShellEvaluateRejectsAnAnswerThatIsOnlyInstructions(t *testing.T) {
	submission := "# Count the lines\n#\n# Write: the command line that does this\n\n"

	result, err := NewShellExecutor().Evaluate(context.Background(), countingExercise(), submission)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != CommandError {
		t.Fatalf("Evaluate() = %q, want %q", result.Status, CommandError)
	}
	if !strings.Contains(result.Output, "did not write a command") {
		t.Errorf("output does not say the answer was empty:\n%s", result.Output)
	}
}

// The instruction header stays in the file the user saves, so it must not be
// mistaken for part of the answer.
func TestShellEvaluateIgnoresTheInstructionHeader(t *testing.T) {
	submission := "# Count the lines in notes.txt\n\nwc -l < notes.txt\n"

	result, err := NewShellExecutor().Evaluate(context.Background(), countingExercise(), submission)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != Success {
		t.Fatalf("Evaluate() = %q, want %q\n%s", result.Status, Success, result.Output)
	}
}

func TestShellEvaluateRequiresTheCommandTheExerciseIsAbout(t *testing.T) {
	ex := countingExercise()
	ex.RequiredCommands = []string{"wc"}

	result, err := NewShellExecutor().Evaluate(context.Background(), ex, "grep -c '' notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != MissingCommand {
		t.Fatalf("Evaluate() = %q, want %q\n%s", result.Status, MissingCommand, result.Output)
	}
	if !strings.Contains(result.Output, "wc") {
		t.Errorf("output does not name the required command:\n%s", result.Output)
	}
}

func TestShellEvaluateAcceptsARequiredCommandGivenByPath(t *testing.T) {
	ex := countingExercise()
	ex.RequiredCommands = []string{"wc"}

	result, err := NewShellExecutor().Evaluate(context.Background(), ex, "/usr/bin/wc -l < notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != Success {
		t.Fatalf("Evaluate() = %q, want %q\n%s", result.Status, Success, result.Output)
	}
}

// The check script decides; a command that reports failure is free to be the
// right answer, as grep's "no matches" exit status is.
func TestShellEvaluateAcceptsACommandThatExitsNonZero(t *testing.T) {
	ex := exercise.Exercise{
		ID:       "sh-002",
		Language: exercise.LanguageShell,
		Setup:    "printf 'nothing here\\n' > notes.txt\n",
		Check:    "[ ! -s \"$CMM_STDOUT\" ] && [ \"$CMM_EXIT\" = 1 ]\n",
	}

	result, err := NewShellExecutor().Evaluate(context.Background(), ex, "grep missing notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != Success {
		t.Fatalf("Evaluate() = %q, want %q\n%s", result.Status, Success, result.Output)
	}
}

// The captured output lives outside the working directory so that a command
// which lists or globs that directory does not see the grader's own files.
func TestShellEvaluateKeepsCapturedOutputOutOfTheWorkingDirectory(t *testing.T) {
	ex := exercise.Exercise{
		ID:       "sh-003",
		Language: exercise.LanguageShell,
		Setup:    "printf 'x\\n' > only.txt\n",
		Check:    "[ \"$(cat \"$CMM_STDOUT\")\" = only.txt ]\n",
	}

	result, err := NewShellExecutor().Evaluate(context.Background(), ex, "ls")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != Success {
		t.Fatalf("Evaluate() = %q, want %q\n%s", result.Status, Success, result.Output)
	}
}

func TestShellEvaluateStopsACommandThatNeverFinishes(t *testing.T) {
	executor := ShellExecutor{timeout: 200 * time.Millisecond}

	result, err := executor.Evaluate(context.Background(), countingExercise(), "sleep 60")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != CommandError {
		t.Fatalf("Evaluate() = %q, want %q", result.Status, CommandError)
	}
	if !strings.Contains(result.Output, "still running") {
		t.Errorf("output does not explain that the command was stopped:\n%s", result.Output)
	}
}

func TestShellPreviewDescribesTheSeededDirectory(t *testing.T) {
	ex := exercise.Exercise{
		ID:       "sh-004",
		Language: exercise.LanguageShell,
		Setup:    "mkdir -p var/log\nprintf 'started\\n' > var/log/system.log\n",
		Check:    "true\n",
	}

	preview, err := NewShellExecutor().Preview(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"var/", "var/log/system.log", "started"} {
		if !strings.Contains(preview, want) {
			t.Errorf("preview does not mention %q:\n%s", want, preview)
		}
	}
}

func TestShellPreviewIsEmptyWithoutSetup(t *testing.T) {
	ex := exercise.Exercise{ID: "sh-005", Language: exercise.LanguageShell, Check: "true\n"}

	preview, err := NewShellExecutor().Preview(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	if preview != "" {
		t.Fatalf("Preview() = %q, want nothing for an exercise that starts from an empty directory", preview)
	}
}

func TestShellVerifyAuthoringAcceptsASoundExercise(t *testing.T) {
	problems, err := NewShellExecutor().VerifyAuthoring(context.Background(), countingExercise())
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("VerifyAuthoring() = %v, want no problems", problems)
	}
}

func TestShellVerifyAuthoringRequiresASolution(t *testing.T) {
	ex := countingExercise()
	ex.Solution = ""

	problems, err := NewShellExecutor().VerifyAuthoring(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "no solution") {
		t.Fatalf("VerifyAuthoring() = %v, want it to ask for a solution", problems)
	}
}

// A check a do-nothing command already satisfies verifies nothing, the same
// way hidden tests a blank stub passes verify nothing.
func TestShellVerifyAuthoringRejectsACheckThatPassesADoNothingCommand(t *testing.T) {
	ex := countingExercise()
	ex.Check = "true\n"
	ex.Solution = "wc -l < notes.txt\n"

	problems, err := NewShellExecutor().VerifyAuthoring(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "does nothing") {
		t.Fatalf("VerifyAuthoring() = %v, want it to reject a check that verifies nothing", problems)
	}
}

func TestShellVerifyAuthoringReportsABrokenSetup(t *testing.T) {
	ex := countingExercise()
	ex.Setup = "cp nowhere/at/all here\n"

	problems, err := NewShellExecutor().VerifyAuthoring(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "setup script failed") {
		t.Fatalf("VerifyAuthoring() = %v, want it to report the failing setup", problems)
	}
}

func TestStripCommentLines(t *testing.T) {
	cases := []struct {
		name       string
		submission string
		want       string
	}{
		{"header only", "# a\n#\n# b\n", ""},
		{"header and command", "# a\n\nls -l\n", "ls -l"},
		{"hash inside a command is an argument", "grep '#' notes.txt", "grep '#' notes.txt"},
		{"indented comment", "   # a\nls", "ls"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := stripCommentLines(c.submission); got != c.want {
				t.Fatalf("stripCommentLines() = %q, want %q", got, c.want)
			}
		})
	}
}
