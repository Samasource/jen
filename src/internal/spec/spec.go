package spec

import (
	"fmt"
	"path"

	"github.com/Samasource/jen/src/internal/constant"
	"github.com/Samasource/jen/src/internal/exec"
	"github.com/Samasource/jen/src/internal/steps"
	"github.com/Samasource/jen/src/internal/steps/choice"
	"github.com/Samasource/jen/src/internal/steps/do"
	execstep "github.com/Samasource/jen/src/internal/steps/exec"
	"github.com/Samasource/jen/src/internal/steps/input"
	"github.com/Samasource/jen/src/internal/steps/option"
	"github.com/Samasource/jen/src/internal/steps/options"
	"github.com/Samasource/jen/src/internal/steps/render"
	"github.com/kylelemons/go-gypsy/yaml"
)

// Spec represents a template specification file found in the root of the template's dir
type Spec struct {
	Name        string
	Description string
	Version     string
	Actions     map[string]Action
}

// Load loads spec object from a template directory
func Load(templateDir string) (*Spec, error) {
	specFilePath := path.Join(templateDir, constant.SpecFileName)
	yamlFile, err := yaml.ReadFile(specFilePath)
	if err != nil {
		return nil, err
	}

	_map, ok := yamlFile.Root.(yaml.Map)
	if !ok {
		return nil, fmt.Errorf("spec file root is expected to be an object")
	}
	return loadFromMap(_map, templateDir)
}

func loadFromMap(_map yaml.Map, templateDir string) (*Spec, error) {
	spec := new(Spec)

	// Load metadata
	metadata, err := getRequiredMap(_map, "metadata")
	if err != nil {
		return nil, err
	}
	spec.Name, err = getRequiredStringFromMap(metadata, "name")
	if err != nil {
		return nil, err
	}
	spec.Description, err = getRequiredStringFromMap(metadata, "description")
	if err != nil {
		return nil, err
	}
	spec.Version, err = getRequiredStringFromMap(metadata, "version")
	if err != nil {
		return nil, err
	}
	if spec.Version != constant.SpecFileVersion {
		return nil, fmt.Errorf("unsupported spec file version %s (expected %s)", spec.Version, constant.SpecFileVersion)
	}

	// Load actions
	actions, err := getRequiredMap(_map, "actions")
	if err != nil {
		return nil, err
	}
	spec.Actions, err = loadActions(actions, templateDir)
	if err != nil {
		return nil, err
	}

	return spec, nil
}

func loadActions(node yaml.Map, templateDir string) (ActionMap, error) {
	var actions []Action
	for name, value := range node {
		stepList, ok := value.(yaml.List)
		if !ok {
			return nil, fmt.Errorf("value of action %q must be a list", name)
		}
		executables, err := loadExecutables(stepList, templateDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load action %q: %w", name, err)
		}
		actions = append(actions, Action{Name: name, Steps: executables})
	}

	// Convert to map
	m := make(ActionMap)
	for _, action := range actions {
		m[action.Name] = action
	}
	return m, nil
}

func loadExecutables(list yaml.List, templateDir string) (exec.Executables, error) {
	var executables exec.Executables
	for idx, value := range list {
		step, err := loadExecutable(value, templateDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load step #%d: %w", idx+1, err)
		}
		executables = append(executables, step)
	}
	return executables, nil
}

func loadExecutable(node yaml.Node, templateDir string) (exec.Executable, error) {
	// Special case for if step
	_map, ok := node.(yaml.Map)
	if ok {
		_, ok = _map["if"]
		if ok {
			return loadIfStep(_map, templateDir)
		}
	}

	// Other steps
	items := []struct {
		name          string
		defaultSubKey string
		fct           func(node yaml.Map, templateDir string) (exec.Executable, error)
	}{
		{
			name: "input",
			fct:  loadInputStep,
		},
		{
			name: "option",
			fct:  loadOptionStep,
		},
		{
			name: "options",
			fct:  loadOptionsStep,
		},
		{
			name: "choice",
			fct:  loadChoiceStep,
		},
		{
			name:          "render",
			defaultSubKey: "source",
			fct:           loadRenderStep,
		},
		{
			name:          "exec",
			defaultSubKey: "commands",
			fct:           loadExecStep,
		},
		{
			name:          "do",
			defaultSubKey: "action",
			fct:           loadDoStep,
		},
	}

	for _, x := range items {
		_map, ok, err := getOptionalMapOrRawStringOrRawStrings(node, x.name, x.defaultSubKey)
		if err != nil {
			return nil, err
		}
		if ok {
			return x.fct(_map, templateDir)
		}
	}

	return nil, fmt.Errorf("unknown step type")
}

func loadIfStep(_map yaml.Map, templateDir string) (exec.Executable, error) {
	condition, err := getRequiredStringFromMap(_map, "if")
	if err != nil {
		return nil, err
	}
	list, err := getRequiredList(_map, "then")
	if err != nil {
		return nil, err
	}
	executables, err := loadExecutables(list, templateDir)
	if err != nil {
		return nil, err
	}
	return steps.If{
		Condition: condition,
		Then:      executables,
	}, nil
}

func loadInputStep(_map yaml.Map, templateDir string) (exec.Executable, error) {
	question, err := getRequiredStringFromMap(_map, "question")
	if err != nil {
		return nil, err
	}
	variable, err := getRequiredStringFromMap(_map, "var")
	if err != nil {
		return nil, err
	}
	defaultValue, err := getOptionalStringFromMap(_map, "default", "")
	if err != nil {
		return nil, err
	}
	return input.Prompt{
		Message: question,
		Var:     variable,
		Default: defaultValue,
	}, nil
}

func loadOptionStep(_map yaml.Map, templateDir string) (exec.Executable, error) {
	question, err := getRequiredStringFromMap(_map, "question")
	if err != nil {
		return nil, err
	}
	variable, err := getRequiredStringFromMap(_map, "var")
	if err != nil {
		return nil, err
	}
	defaultValue, err := getOptionalBool(_map, "default", false)
	if err != nil {
		return nil, err
	}
	return option.Prompt{
		Message: question,
		Var:     variable,
		Default: defaultValue,
	}, nil
}

func loadOptionsStep(_map yaml.Map, templateDir string) (exec.Executable, error) {
	question, err := getRequiredStringFromMap(_map, "question")
	if err != nil {
		return nil, err
	}

	// Load children
	list, err := getRequiredList(_map, "items")
	if err != nil {
		return nil, err
	}
	var items []options.Item
	for _, child := range list {
		childMap, ok := child.(yaml.Map)
		if !ok {
			return nil, fmt.Errorf("items of %q property must be objects", "options")
		}
		question, err := getRequiredStringFromMap(childMap, "text")
		if err != nil {
			return nil, err
		}
		variable, err := getRequiredStringFromMap(childMap, "var")
		if err != nil {
			return nil, err
		}
		defaultValue, err := getOptionalBool(childMap, "default", false)
		if err != nil {
			return nil, err
		}
		items = append(items, options.Item{
			Text:    question,
			Var:     variable,
			Default: defaultValue,
		})
	}

	return options.Prompt{
		Message: question,
		Items:   items,
	}, nil
}

func loadChoiceStep(_map yaml.Map, templateDir string) (exec.Executable, error) {
	question, err := getRequiredStringFromMap(_map, "question")
	if err != nil {
		return nil, err
	}
	variable, err := getRequiredStringFromMap(_map, "var")
	if err != nil {
		return nil, err
	}
	defaultValue, err := getOptionalStringFromMap(_map, "default", "")
	if err != nil {
		return nil, err
	}

	// Load children
	list, err := getRequiredList(_map, "items")
	if err != nil {
		return nil, err
	}
	var items []choice.Item
	for _, child := range list {
		childMap, ok := child.(yaml.Map)
		if !ok {
			return nil, fmt.Errorf("items of %q property must be objects", "options")
		}
		text, err := getRequiredStringFromMap(childMap, "text")
		if err != nil {
			return nil, err
		}
		value, err := getRequiredStringFromMap(childMap, "value")
		if err != nil {
			return nil, err
		}
		items = append(items, choice.Item{
			Text:  text,
			Value: value,
		})
	}

	return choice.Prompt{
		Message: question,
		Var:     variable,
		Default: defaultValue,
		Items:   items,
	}, nil
}

func loadRenderStep(_map yaml.Map, templateDir string) (exec.Executable, error) {
	source, err := getRequiredStringFromMap(_map, "source")
	if err != nil {
		return nil, err
	}

	return render.Render{
		InputDir: path.Join(templateDir, source),
	}, nil
}

func loadExecStep(_map yaml.Map, templateDir string) (exec.Executable, error) {
	commands, err := getRequiredStringsOrStringFromMap(_map, "commands")
	if err != nil {
		return nil, err
	}

	return execstep.Exec{
		Commands: commands,
	}, nil
}

func loadDoStep(_map yaml.Map, templateDir string) (exec.Executable, error) {
	action, err := getRequiredStringFromMap(_map, "action")
	if err != nil {
		return nil, err
	}

	return do.Do{
		Action: action,
	}, nil
}