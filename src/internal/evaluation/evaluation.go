package evaluation

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
)

// Context encapsulates everything required for template evaluation and rendering
type Context interface {
	// GetEvalVars returns a dictionary of the project's variable names mapped to
	// their corresponding values for evaluation purposes. It does not include the
	// process' env var.
	GetEvalVars() map[string]interface{}

	// GetPlaceholders returns a map of special placeholders that can be used instead
	// of go template expressions, for more lightweight templating, especially for the
	// project's name, which appears everywhere.
	GetPlaceholders() map[string]string

	// GetShellVars returns all env vars to be used when invoking shell commands,
	// including the current process' env vars, the project's vars and an augmented
	// PATH var including extra bin dirs.
	GetShellVars(includeProcessVars bool) []string
}

// RenderMode determines how/if rendering enabled/disabled state should change for an item
// and all its children recursively, compared to parent's state
type RenderMode int

const (
	// DefaultRendering preserves current rendering mode of parent
	DefaultMode RenderMode = iota

	// TemplateRendering enables template rendering for itself and all children recursively
	TemplateMode

	// CopyRendering disables template rendering for itself and all children recursively
	CopyMode

	// InsertRendering enables template insertion, but only for a single file
	InsertMode
)

// EvalBoolExpression determines whether given go template expression evaluates to true or false
func EvalBoolExpression(context Context, expression string) (bool, error) {
	ifExpr := "{{if " + expression + "}}true{{end}}"
	result, err := EvalTemplate(context, ifExpr)
	if err != nil {
		return false, fmt.Errorf("evaluate expression %q: %w", expression, err)
	}
	return result == "true", nil
}

// EvalTemplate interpolates given template text into a final output string
func EvalTemplate(context Context, text string) (string, error) {
	// Escape triple braces
	doubleOpen := strings.Repeat("{", 2)
	doubleClose := strings.Repeat("}", 2)
	tripleOpen := strings.Repeat("{", 3)
	tripleClose := strings.Repeat("}", 3)
	text = strings.ReplaceAll(text, tripleOpen, doubleOpen+"`"+doubleOpen+"`"+doubleClose)
	text = strings.ReplaceAll(text, tripleClose, doubleOpen+"`"+doubleClose+"`"+doubleClose)

	// Perform replacement of placeholders
	for placeholderName, placeholderValue := range context.GetPlaceholders() {
		text = strings.ReplaceAll(text, placeholderName, placeholderValue)
	}

	// Render go template
	tmpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(text)
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", text, err)
	}
	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, context.GetEvalVars())
	if err != nil {
		return "", fmt.Errorf("evaluate template %q: %w", text, err)
	}
	return buffer.String(), nil
}

var doubleBracketRegexp = regexp.MustCompile(`\[\[.*]]`)

// evalFileName interpolates the double-brace expressions, evaluates and removes the conditionals in double-bracket
// expressions and returns the final file/dir name and whether it should be included in output and whether it should be
// rendered.
func evalFileName(context Context, name string) (string, bool, RenderMode, error) {
	// Double-bracket expressions (ie: "[[.option]]") in names are evaluated to determine whether the file/folder should be
	// included in output and that expression then gets stripped from the name
	for {
		// Find expression
		loc := doubleBracketRegexp.FindStringIndex(name)
		if loc == nil {
			break
		}
		exp := name[loc[0]+2 : loc[1]-2]

		// Evaluate expression
		value, err := EvalBoolExpression(context, exp)
		if err != nil {
			return "", false, DefaultMode, fmt.Errorf("failed to eval double-bracket expression in name %q: %w", name, err)
		}

		// Should we exclude file/folder?
		if !value {
			return "", false, DefaultMode, nil
		}

		// Remove expression from name
		name = name[:loc[0]] + name[loc[1]:]
	}

	// Double-brace expressions (ie: "{{.name}}") in names get interpolated as expected
	outputName, err := EvalTemplate(context, name)
	if err != nil {
		return "", false, DefaultMode, fmt.Errorf("failed to evaluate double-brace expression in name %q: %w", name, err)
	}

	// Determine render mode and remove .tmpl/.notmpl extensions
	renderMode, outputName := getRenderModeAndRemoveExtension(outputName)
	return outputName, true, renderMode, nil
}

var tmplExtensionRegexp = regexp.MustCompile(`\.tmpl($|\.)`)
var notmplExtensionRegexp = regexp.MustCompile(`\.notmpl($|\.)`)
var insertExtensionRegexp = regexp.MustCompile(`\.insert($|\.)`)

// getRenderModeAndRemoveExtension determines render mode based on .tmpl/.notmpl extensions and removes those extensions
func getRenderModeAndRemoveExtension(name string) (RenderMode, string) {
	name, ok := removeRegexp(name, tmplExtensionRegexp)
	if ok {
		return TemplateMode, name
	}

	name, ok = removeRegexp(name, notmplExtensionRegexp)
	if ok {
		return CopyMode, name
	}

	name, ok = removeRegexp(name, insertExtensionRegexp)
	if ok {
		return InsertMode, name
	}

	return DefaultMode, name
}

func removeRegexp(input string, regexp *regexp.Regexp) (string, bool) {
	output := regexp.ReplaceAllString(input, "$1")
	return output, len(output) != len(input)
}
