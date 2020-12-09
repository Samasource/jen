package model

import (
	. "github.com/Samasource/jen/internal/logging"
)

type Action struct {
	Name  string
	Steps Executables
}

type ActionMap map[string]Action

func (a Action) String() string {
	return a.Name
}

func (a Action) Execute(config Config) error {
	Log("Executing sub-steps of action %q", a.Name)
	return a.Steps.Execute(config)
}