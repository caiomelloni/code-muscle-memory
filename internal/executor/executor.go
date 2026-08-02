package executor

import (
	"context"
	"fmt"

	"code-muscle-memory/internal/exercise"
)

type Status string

const (
	CompileError   Status = "compile_error"
	TestFailure    Status = "test_failure"
	MutantEscaped  Status = "mutant_escaped"
	MissingFeature Status = "missing_feature"
	CommandError   Status = "command_error"
	CheckFailure   Status = "check_failure"
	MissingCommand Status = "missing_command"
	Success        Status = "success"
)

type Result struct {
	Status Status
	Output string
}

// Executor grades a submission for one language. Evaluate runs the user's
// answer and reports how it did; VerifyAuthoring runs the checks that only the
// exercise's own code can answer, for "cmm validate".
type Executor interface {
	Evaluate(ctx context.Context, ex exercise.Exercise, submission string) (Result, error)
	VerifyAuthoring(ctx context.Context, ex exercise.Exercise) ([]string, error)
}

// Previewer is implemented by executors whose exercises start from a prepared
// environment the user has to see to answer at all — a shell exercise's seeded
// directory, for instance. The returned text is shown with the instructions.
type Previewer interface {
	Preview(ctx context.Context, ex exercise.Exercise) (string, error)
}

// For returns the executor that grades exercises in the given language.
func For(language string) (Executor, error) {
	switch language {
	case exercise.LanguageGo:
		return NewGoExecutor(), nil
	case exercise.LanguageShell:
		return NewShellExecutor(), nil
	default:
		return nil, fmt.Errorf("no executor for language %q", language)
	}
}
