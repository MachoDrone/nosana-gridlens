package execx

import (
	"context"
	"fmt"
	"strings"
)

type FakeRunner struct {
	Paths    map[string]string
	Results  map[string]Result
	Commands []Command
}

func NewFakeRunner() *FakeRunner {
	return &FakeRunner{
		Paths:   map[string]string{},
		Results: map[string]Result{},
	}
}

func (f *FakeRunner) SetPath(name string, path string) {
	f.Paths[name] = path
}

func (f *FakeRunner) SetResult(name string, args []string, result Result) {
	f.Results[f.key(name, args)] = result
}

func (f *FakeRunner) LookPath(file string) (string, error) {
	path, ok := f.Paths[file]
	if !ok {
		return "", fmt.Errorf("%s: command not found", file)
	}
	return path, nil
}

func (f *FakeRunner) Run(_ context.Context, name string, args ...string) Result {
	copiedArgs := append([]string(nil), args...)
	f.Commands = append(f.Commands, Command{Name: name, Args: copiedArgs})

	result, ok := f.Results[f.key(name, args)]
	if !ok {
		return MissingCommandResult(name)
	}
	return result
}

func (f *FakeRunner) key(name string, args []string) string {
	if len(args) == 0 {
		return name
	}
	return name + "\x00" + strings.Join(args, "\x00")
}
