package executor

import (
	"context"
	"strings"
	"testing"
)

func TestVerifyAuthoringAcceptsSoundImplementation(t *testing.T) {
	ex := addExercise()
	ex.StarterCode = "package exercise\n\nfunc Add(a, b int) int {\n\treturn 0\n}\n"
	ex.Solution = "package exercise\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n"

	problems, err := NewGoExecutor().VerifyAuthoring(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("problems = %q, want none", problems)
	}
}

func TestVerifyAuthoringFlagsMissingSolution(t *testing.T) {
	ex := addExercise()

	problems, err := NewGoExecutor().VerifyAuthoring(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "no solution") {
		t.Fatalf("problems = %q, want one about the missing solution", problems)
	}
}

func TestVerifyAuthoringFlagsFailingSolution(t *testing.T) {
	ex := addExercise()
	ex.Solution = "package exercise\n\nfunc Add(a, b int) int {\n\treturn a - b\n}\n"

	problems, err := NewGoExecutor().VerifyAuthoring(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "does not pass") {
		t.Fatalf("problems = %q, want one about the failing solution", problems)
	}
}

func TestVerifyAuthoringFlagsVacuousTests(t *testing.T) {
	ex := addExercise()
	ex.Tests = "package exercise\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tAdd(2, 3)\n}\n"
	ex.StarterCode = "package exercise\n\nfunc Add(a, b int) int {\n\treturn 0\n}\n"
	ex.Solution = "package exercise\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n"

	problems, err := NewGoExecutor().VerifyAuthoring(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "verify nothing") {
		t.Fatalf("problems = %q, want one about vacuous tests", problems)
	}
}

func TestVerifyAuthoringAcceptsSoundTestWriting(t *testing.T) {
	ex := signExercise()

	problems, err := NewGoExecutor().VerifyAuthoring(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 0 {
		t.Fatalf("problems = %q, want none", problems)
	}
}

func TestVerifyAuthoringFlagsBrokenMutants(t *testing.T) {
	ex := signExercise()
	ex.Mutants = []string{
		ex.SubjectCode,
		"package exercise\n\nfunc Sign(n int) string {",
	}
	ex.MutantHints = nil

	problems, err := NewGoExecutor().VerifyAuthoring(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 2 {
		t.Fatalf("problems = %q, want two", problems)
	}
	if !strings.Contains(problems[0], "mutant 0 is identical") {
		t.Fatalf("problems[0] = %q, want identical-mutant problem", problems[0])
	}
	if !strings.Contains(problems[1], "mutant 1 does not compile") {
		t.Fatalf("problems[1] = %q, want non-compiling-mutant problem", problems[1])
	}
}

func TestVerifyAuthoringFlagsNonCompilingSubject(t *testing.T) {
	ex := signExercise()
	ex.SubjectCode = "package exercise\n\nfunc Sign(n int) string {"

	problems, err := NewGoExecutor().VerifyAuthoring(context.Background(), ex)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) == 0 || !strings.Contains(problems[0], "subject code does not compile") {
		t.Fatalf("problems = %q, want one about the subject not compiling", problems)
	}
}
