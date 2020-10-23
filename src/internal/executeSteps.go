package internal

import (
	"fmt"
)

type Executable interface {
	Execute(context Context) error
}

func (root Spec) Execute(context *Context) error {
	return execute(context, root.Steps)
}

func execute(context *Context, steps []*StepUnion) error {
	for i, step := range steps {
		if err := step.execute(context, i + 1); err != nil {
			return err
		}
	}
	return nil
}

func (step StepUnion) execute(context *Context, index int) error {
	if step.If != "" {
		result, err := EvalBoolExpression(*context, step.If)
		if err != nil {
			return fmt.Errorf("evaluate step #%d conditional expression: %w", index, err)
		}
		if !result {
			Logf("Skipping step #%d because conditional %q evaluates to false", index, step.If)
			return nil
		}
	}

	var err error
	switch {
	case step.String != nil:
		err = step.String.Execute(*context)
	case step.Option != nil:
		err = step.Option.Execute(*context)
	case step.Multi != nil:
		err = step.Multi.Execute(*context)
	case step.Select != nil:
		err = step.Select.Execute(*context)
	case step.SetOutput != "":
		err = setOutput(context, step.SetOutput)
	case step.Render != "":
		err = render(*context, step.Render)
	case step.Do != "":
		err = do(context, step.Do)
	case step.Exec != "":
		err = execShell(*context, step.Exec)
	default:
		return fmt.Errorf("unsupported step #%d", index)
	}
	return err
}