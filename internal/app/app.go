package app

import (
	"bufio"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"code-muscle-memory/internal/executor"
	"code-muscle-memory/internal/exercise"
	"code-muscle-memory/internal/scheduler"
	"code-muscle-memory/internal/storage"
)

type Config struct {
	ExerciseDir  string
	ProgressPath string
	Stdout       io.Writer
	Stderr       io.Writer
	Stdin        io.Reader
	Now          func() time.Time
}

type App struct {
	cfg       Config
	store     storage.Store
	scheduler scheduler.Scheduler
	executor  executor.GoExecutor
}

func New(cfg Config) App {
	if cfg.Stdout == nil {
		cfg.Stdout = io.Discard
	}
	if cfg.Stderr == nil {
		cfg.Stderr = io.Discard
	}
	if cfg.Stdin == nil {
		cfg.Stdin = os.Stdin
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return App{
		cfg:       cfg,
		store:     storage.NewJSONStore(cfg.ProgressPath),
		scheduler: scheduler.NewSM2(),
		executor:  executor.NewGoExecutor(),
	}
}

func (a App) Next(ctx context.Context) error {
	exercises, progress, err := a.load()
	if err != nil {
		return err
	}

	now := a.cfg.Now()
	ex, ok := chooseNext(exercises, progress, now)
	if !ok {
		fmt.Fprintln(a.cfg.Stdout, "No exercises available.")
		return nil
	}
	item := progress.Items[ex.ID]
	item.ExerciseID = ex.ID
	if !item.DueAt.IsZero() && item.DueAt.After(now) {
		fmt.Fprintf(a.cfg.Stdout, "Next review is not due yet. Earliest: %s\n", item.DueAt.Format(time.RFC1123))
		return nil
	}
	progress.CurrentExerciseID = ex.ID
	if err := a.store.Save(progress); err != nil {
		return err
	}

	result, solution, err := a.runExercise(ctx, ex, progress.Attempts[ex.ID])
	if err != nil {
		return err
	}
	if result.Status != executor.Success {
		return a.saveAttempt(progress, ex.ID, solution)
	}

	rating := a.promptRating(now, item)
	delete(progress.Attempts, ex.ID)
	progress.CurrentExerciseID = ""
	progress.Items[ex.ID] = a.scheduler.Review(now, item, rating)
	return a.store.Save(progress)
}

// Try runs a single exercise as practice: it opens the editor and evaluates
// the solution like a review would, but never reads or writes stored
// progress, so the review schedule is unaffected.
func (a App) Try(ctx context.Context, id string) error {
	exercises, err := exercise.LoadDir(a.cfg.ExerciseDir)
	if err != nil {
		return err
	}
	ex, ok := findExercise(exercises, id)
	if !ok {
		return fmt.Errorf("exercise %q not found", id)
	}

	fmt.Fprintln(a.cfg.Stdout, "Practice run: results will not affect your review progress.")
	_, _, err = a.runExercise(ctx, ex, "")
	return err
}

// runExercise opens the exercise in the editor, evaluates the solution, and
// prints the outcome. It never touches stored progress.
func (a App) runExercise(ctx context.Context, ex exercise.Exercise, previousAttempt string) (executor.Result, string, error) {
	solutionPath, cleanup, err := writeStarter(ex, previousAttempt)
	if err != nil {
		return executor.Result{}, "", err
	}
	defer cleanup()

	a.printExercise(ex, solutionPath, previousAttempt != "")
	if err := openEditor(ctx, solutionPath); err != nil {
		return executor.Result{}, "", err
	}

	solution, err := os.ReadFile(solutionPath)
	if err != nil {
		return executor.Result{}, "", err
	}
	result, err := a.executor.Evaluate(ctx, ex, string(solution))
	if err != nil {
		return executor.Result{}, "", err
	}

	switch result.Status {
	case executor.Success:
		fmt.Fprintln(a.cfg.Stdout, "\nPASS")
		if strings.TrimSpace(result.Output) != "" {
			fmt.Fprintln(a.cfg.Stdout, result.Output)
		}
	case executor.CompileError:
		fmt.Fprintln(a.cfg.Stdout, "\nCOMPILE ERROR")
		fmt.Fprintln(a.cfg.Stdout, result.Output)
	case executor.TestFailure:
		fmt.Fprintln(a.cfg.Stdout, "\nTEST FAILURE")
		fmt.Fprintln(a.cfg.Stdout, result.Output)
	case executor.MutantEscaped:
		fmt.Fprintln(a.cfg.Stdout, "\nINCOMPLETE TEST")
		fmt.Fprintln(a.cfg.Stdout, result.Output)
	case executor.MissingFeature:
		fmt.Fprintln(a.cfg.Stdout, "\nMISSING STRUCTURE")
		fmt.Fprintln(a.cfg.Stdout, result.Output)
	}
	return result, string(solution), nil
}

// saveAttempt persists the user's unsuccessful submission so it can be
// restored the next time they retry this exercise, instead of starting over
// from the blank instructions template.
func (a App) saveAttempt(progress storage.ProgressFile, exerciseID, solution string) error {
	if progress.Attempts == nil {
		progress.Attempts = map[string]string{}
	}
	progress.Attempts[exerciseID] = solution
	return a.store.Save(progress)
}

func (a App) List(ctx context.Context) error {
	_ = ctx
	exercises, progress, err := a.load()
	if err != nil {
		return err
	}
	now := a.cfg.Now()
	for _, ex := range exercises {
		item, ok := progress.Items[ex.ID]
		fmt.Fprintf(a.cfg.Stdout, "%-22s %-12s difficulty=%d topic=%s\n", ex.ID, reviewStatus(item, ok, now), ex.Difficulty, ex.Topic)
	}
	return nil
}

func (a App) Stats(ctx context.Context) error {
	_ = ctx
	exercises, progress, err := a.load()
	if err != nil {
		return err
	}
	now := a.cfg.Now()
	newCount := 0
	learning := 0
	due := 0
	for _, ex := range exercises {
		item, ok := progress.Items[ex.ID]
		if !ok {
			newCount++
			continue
		}
		if state := item.CurrentState(); state == scheduler.StateLearning || state == scheduler.StateRelearning {
			learning++
			continue
		}
		if !item.DueAt.After(now) {
			due++
		}
	}
	fmt.Fprintf(a.cfg.Stdout, "Exercises: %d\nNew: %d\nLearning: %d\nDue now: %d\n", len(exercises), newCount, learning, due)
	return nil
}

// Describe prints an exercise's full instructions and review status without
// opening it in $EDITOR, so a user can preview what an exercise asks for.
func (a App) Describe(ctx context.Context, id string) error {
	_ = ctx
	exercises, progress, err := a.load()
	if err != nil {
		return err
	}
	ex, ok := findExercise(exercises, id)
	if !ok {
		return fmt.Errorf("exercise %q not found", id)
	}

	item, hasProgress := progress.Items[ex.ID]
	fmt.Fprintf(a.cfg.Stdout, "# %s (%s)\n\n%s\n", ex.Title, ex.ID, ex.Description)
	if strings.TrimSpace(ex.Objective) != "" {
		fmt.Fprintf(a.cfg.Stdout, "\nObjective: %s\n", ex.Objective)
	}
	fmt.Fprintf(a.cfg.Stdout, "Topic: %s | Difficulty: %d | Kind: %s | Status: %s\n", ex.Topic, ex.Difficulty, ex.EffectiveKind(), reviewStatus(item, hasProgress, a.cfg.Now()))

	fmt.Fprintln(a.cfg.Stdout)
	for _, instruction := range kindInstructions(ex) {
		fmt.Fprintln(a.cfg.Stdout, instruction)
	}
	return nil
}

// Validate loads every exercise and runs the deep authoring checks that
// schema validation cannot: solutions must pass their hidden tests, starter
// code must not, and test-writing subjects and mutants must compile. It is
// meant for authors adding or revising exercises, not for regular practice.
func (a App) Validate(ctx context.Context) error {
	exercises, err := exercise.LoadDir(a.cfg.ExerciseDir)
	if err != nil {
		return err
	}

	failures := 0
	for _, ex := range exercises {
		problems, err := a.executor.VerifyAuthoring(ctx, ex)
		if err != nil {
			return err
		}
		if len(problems) == 0 {
			fmt.Fprintf(a.cfg.Stdout, "ok   %s\n", ex.ID)
			continue
		}
		failures++
		fmt.Fprintf(a.cfg.Stdout, "FAIL %s\n", ex.ID)
		for _, problem := range problems {
			fmt.Fprintf(a.cfg.Stdout, "  - %s\n", strings.ReplaceAll(problem, "\n", "\n    "))
		}
	}

	if failures > 0 {
		return fmt.Errorf("%d of %d exercises failed validation", failures, len(exercises))
	}
	fmt.Fprintf(a.cfg.Stdout, "All %d exercises passed validation.\n", len(exercises))
	return nil
}

func reviewStatus(item scheduler.Progress, hasProgress bool, now time.Time) string {
	if !hasProgress {
		return "new"
	}
	if state := item.CurrentState(); state == scheduler.StateLearning || state == scheduler.StateRelearning {
		if item.DueAt.After(now) {
			return string(state) + " (due " + item.DueAt.Format("15:04") + ")"
		}
		return string(state) + " (due now)"
	}
	if item.DueAt.After(now) {
		return "due " + item.DueAt.Format("2006-01-02")
	}
	return "due now"
}

func findExercise(exercises []exercise.Exercise, id string) (exercise.Exercise, bool) {
	for _, ex := range exercises {
		if ex.ID == id {
			return ex, true
		}
	}
	return exercise.Exercise{}, false
}

func (a App) load() ([]exercise.Exercise, storage.ProgressFile, error) {
	exercises, err := exercise.LoadDir(a.cfg.ExerciseDir)
	if err != nil {
		return nil, storage.ProgressFile{}, err
	}
	progress, err := a.store.Load()
	if err != nil {
		return nil, storage.ProgressFile{}, err
	}
	return exercises, progress, nil
}

func (a App) printExercise(ex exercise.Exercise, solutionPath string, restoredAttempt bool) {
	fmt.Fprintf(a.cfg.Stdout, "\n# %s\n\n%s\n\nObjective: %s\nTopic: %s | Difficulty: %d\n\nEditing: %s\n", ex.Title, ex.Description, ex.Objective, ex.Topic, ex.Difficulty, solutionPath)
	if restoredAttempt {
		fmt.Fprintln(a.cfg.Stdout, "(restored your previous attempt)")
	}
}

func (a App) promptRating(now time.Time, item scheduler.Progress) scheduler.Rating {
	previews := a.scheduler.Preview(now, item)
	reader := bufio.NewReader(a.cfg.Stdin)
	for {
		fmt.Fprintf(a.cfg.Stdout, "Rate this review [a]gain %s / [h]ard %s / [g]ood %s / [e]asy %s: ",
			formatInterval(previews[scheduler.Again]),
			formatInterval(previews[scheduler.Hard]),
			formatInterval(previews[scheduler.Good]),
			formatInterval(previews[scheduler.Easy]))
		text, err := reader.ReadString('\n')
		input := strings.ToLower(strings.TrimSpace(text))
		switch input {
		case "good", "g":
			return scheduler.Good
		case "again", "a":
			return scheduler.Again
		case "easy", "e":
			return scheduler.Easy
		case "hard", "h":
			return scheduler.Hard
		}
		if err != nil {
			// Input ran out (e.g. non-interactive stdin); nobody can answer a
			// re-prompt, so record the middle-of-the-road rating.
			return scheduler.Good
		}
		if input == "" {
			fmt.Fprintln(a.cfg.Stdout, "A rating is required. Enter again, hard, good, or easy.")
			continue
		}
		fmt.Fprintf(a.cfg.Stdout, "Unknown rating %q. Enter again, hard, good, or easy.\n", input)
	}
}

// formatInterval renders a preview delay the way Anki labels its answer
// buttons: minutes for intra-day steps, whole days otherwise.
func formatInterval(d time.Duration) string {
	if d < 24*time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
}

func chooseNext(exercises []exercise.Exercise, progress storage.ProgressFile, now time.Time) (exercise.Exercise, bool) {
	if len(exercises) == 0 {
		return exercise.Exercise{}, false
	}
	if progress.CurrentExerciseID != "" {
		for _, ex := range exercises {
			if ex.ID == progress.CurrentExerciseID {
				return ex, true
			}
		}
	}

	// Anki queue order: due learning/relearning cards, then due reviews,
	// then new cards.
	var learning, review, fresh []exercise.Exercise
	for _, ex := range exercises {
		item, ok := progress.Items[ex.ID]
		switch {
		case !ok:
			fresh = append(fresh, ex)
		case item.DueAt.After(now):
			continue
		case item.CurrentState() == scheduler.StateReview:
			review = append(review, ex)
		default:
			learning = append(learning, ex)
		}
	}

	byDue := func(list []exercise.Exercise) func(i, j int) bool {
		return func(i, j int) bool {
			left, right := progress.Items[list[i].ID], progress.Items[list[j].ID]
			if !left.DueAt.Equal(right.DueAt) {
				return left.DueAt.Before(right.DueAt)
			}
			return list[i].ID < list[j].ID
		}
	}
	sort.SliceStable(learning, byDue(learning))
	sort.SliceStable(review, byDue(review))
	sort.SliceStable(fresh, func(i, j int) bool {
		if fresh[i].Difficulty != fresh[j].Difficulty {
			return fresh[i].Difficulty < fresh[j].Difficulty
		}
		return fresh[i].ID < fresh[j].ID
	})

	for _, group := range [][]exercise.Exercise{learning, review, fresh} {
		if len(group) > 0 {
			return group[0], true
		}
	}

	// Nothing is due and nothing is new: fall back to the exercise with the
	// earliest upcoming review so the caller can report when it unlocks.
	earliest := exercises[0]
	for _, ex := range exercises[1:] {
		if progress.Items[ex.ID].DueAt.Before(progress.Items[earliest.ID].DueAt) {
			earliest = ex
		}
	}
	return earliest, true
}

func writeStarter(ex exercise.Exercise, previousAttempt string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "cmm-edit-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	path := filepath.Join(dir, "solution.go")
	code := previousAttempt
	if strings.TrimSpace(code) == "" {
		code = starterWithInstructions(ex)
	}
	if strings.TrimSpace(code) == "" {
		code = "package exercise\n"
	}
	if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
}

func starterWithInstructions(ex exercise.Exercise) string {
	lines := []string{
		"// " + ex.Title,
		"//",
		"// What to do: " + ex.Description,
	}
	if strings.TrimSpace(ex.Objective) != "" {
		lines = append(lines, "// Objective: "+ex.Objective)
	}

	for _, instruction := range kindInstructions(ex) {
		lines = append(lines, "//")
		lines = appendCommentBlock(lines, "", instruction)
	}

	lines = append(lines, "", "package exercise", "")
	return strings.Join(lines, "\n")
}

// kindInstructions describes the API the user must work against, phrased
// according to the exercise's kind: what to implement, or what is already
// given plus what test functions to write.
func kindInstructions(ex exercise.Exercise) []string {
	switch ex.EffectiveKind() {
	case exercise.KindTestWriting:
		var instructions []string
		for _, target := range implementationTargets(ex.SubjectCode) {
			instructions = append(instructions, "Given: "+target)
		}
		instructions = append(instructions, "Write: one or more functions whose names start with "+quoted("Test")+", each receiving a parameter called "+quoted("t")+" of type pointer to testing.T and returning nothing")
		for _, feature := range ex.RequiredTestFeatures {
			instructions = append(instructions, "Use: "+featureInstruction(feature))
		}
		return instructions
	default:
		var instructions []string
		for _, target := range implementationTargets(ex.StarterCode) {
			instructions = append(instructions, "Implement: "+target)
		}
		return instructions
	}
}

// featureInstruction phrases a required test feature as prose, describing the
// concept without leaking the syntax the user is meant to recall.
func featureInstruction(feature string) string {
	switch feature {
	case exercise.FeatureSubtests:
		return "named subtests, giving each case its own name"
	case exercise.FeatureTable:
		return "a table of test cases driven by a loop"
	default:
		return feature
	}
}

func implementationTargets(starter string) []string {
	if strings.TrimSpace(starter) == "" {
		return nil
	}

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "starter.go", starter, parser.ParseComments)
	if err != nil {
		return nil
	}

	var targets []string
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok == token.TYPE {
				targets = append(targets, typeDescriptions(d)...)
			}
		case *ast.FuncDecl:
			targets = append(targets, funcDescription(d))
		}
	}
	return targets
}

func typeDescriptions(decl *ast.GenDecl) []string {
	var descriptions []string
	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			descriptions = append(descriptions, "a type called "+quoted(typeSpec.Name.Name))
			continue
		}

		fields := fieldDescriptions(structType.Fields, "field")
		if len(fields) == 0 {
			descriptions = append(descriptions, "a struct type called "+quoted(typeSpec.Name.Name))
			continue
		}
		descriptions = append(descriptions, "a struct type called "+quoted(typeSpec.Name.Name)+" with "+joinEnglish(fields))
	}
	return descriptions
}

func funcDescription(fn *ast.FuncDecl) string {
	subject := "a function called " + quoted(fn.Name.Name)
	if fn.Recv == nil {
		return subject + ", " + paramsDescription(fn.Type.Params) + ", and " + resultsDescription(fn.Type.Results)
	}
	return "a method called " + quoted(fn.Name.Name) + " on " + receiverDescription(fn.Recv) + ", " + paramsDescription(fn.Type.Params) + ", and " + resultsDescription(fn.Type.Results)
}

func receiverDescription(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return "the receiver type"
	}
	return typePhrase(recv.List[0].Type)
}

func paramsDescription(params *ast.FieldList) string {
	fields := fieldDescriptions(params, "parameter")
	if len(fields) == 0 {
		return "receives no parameters"
	}
	return "receives " + joinEnglish(fields)
}

func resultsDescription(results *ast.FieldList) string {
	fields := fieldDescriptions(results, "value")
	if len(fields) == 0 {
		return "returns nothing"
	}
	return "returns " + joinEnglish(fields)
}

func fieldDescriptions(fields *ast.FieldList, namedKind string) []string {
	if fields == nil {
		return nil
	}

	var descriptions []string
	for _, field := range fields.List {
		fieldType := typePhrase(field.Type)
		if len(field.Names) == 0 {
			descriptions = append(descriptions, article(fieldType)+" "+fieldType)
			continue
		}
		for _, name := range field.Names {
			descriptions = append(descriptions, "a "+namedKind+" called "+quoted(name.Name)+" of type "+fieldType)
		}
	}
	return descriptions
}

func quoted(name string) string {
	return "\"" + name + "\""
}

func typePhrase(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "slice of " + typePhrase(t.Elt)
		}
		return "array of " + typePhrase(t.Elt)
	case *ast.StarExpr:
		return "pointer to " + typePhrase(t.X)
	case *ast.SelectorExpr:
		return typePhrase(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map from " + typePhrase(t.Key) + " to " + typePhrase(t.Value)
	case *ast.InterfaceType:
		return "interface"
	case *ast.StructType:
		return "struct"
	default:
		return nodeString(token.NewFileSet(), expr)
	}
}

func joinEnglish(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", and " + items[len(items)-1]
	}
}

func article(phrase string) string {
	lower := strings.ToLower(phrase)
	for _, prefix := range []string{"a", "e", "i", "o", "u"} {
		if strings.HasPrefix(lower, prefix) {
			return "an"
		}
	}
	return "a"
}

func nodeString(fileSet *token.FileSet, node any) string {
	var builder strings.Builder
	_ = printer.Fprint(&builder, fileSet, node)
	return builder.String()
}

func appendCommentBlock(lines []string, prefix, text string) []string {
	for i, line := range strings.Split(text, "\n") {
		if i == 0 {
			lines = append(lines, "// "+prefix+line)
			continue
		}
		lines = append(lines, "// "+line)
	}
	return lines
}

func openEditor(ctx context.Context, path string) error {
	editor := os.Getenv("EDITOR")
	if strings.TrimSpace(editor) == "" {
		editor = "vi"
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", editor+" "+shellQuote(path))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
