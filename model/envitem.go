package model

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

type EnvVar interface {
	GetName() string
	GetValue() string
	GetFileList() *FileList
	Apply(ctx context.Context, logger *slog.Logger, modifiers ...Modifier) error
}

type EnvItem struct {
	Name      string
	Value     string
	Modifiers []string
	FileList  *FileList
}

func (e *EnvItem) GetName() string {
	if e == nil {
		return ""
	}

	return e.Name
}

func (e *EnvItem) GetValue() string {
	if e == nil {
		return ""
	}

	return e.Value
}

func (e *EnvItem) GetFileList() *FileList {
	if e == nil {
		return nil
	}

	return e.FileList
}

func (e *EnvItem) AddFiles(file ...string) {
	if e == nil {
		return
	}

	if e.FileList == nil {
		e.FileList = &FileList{}
	}
	e.FileList = e.FileList.Append(file...)
}

func (e *EnvItem) Apply(ctx context.Context, logger *slog.Logger, modifiers ...Modifier) error {
	if e == nil {
		return errors.New("env item is nil")
	}

	ref := &SecretRef{
		Value: e.Value,
		Files: e.FileList,
	}
	defer func() {
		e.Value = ref.Value
		e.FileList = ref.Files
	}()

	for _, name := range e.Modifiers {
		var found bool
		for _, m := range modifiers {
			if m.Tags().Contains(name) {
				logger.DebugContext(ctx, "applying modifier",
					slog.String("tag", name),
				)
				if err := m.Apply(ctx, ref); err != nil {
					return fmt.Errorf("applying modifier %q: %w", name, err)
				}
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("unknown modifier: %s", name)
		}
	}

	return nil
}
