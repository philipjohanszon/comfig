package comfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

type settings[T any] struct {
	source             Source
	extension          string
	validate           func(config T) error
	parse              func(raw []byte) (T, error)
	resolversFactories []ResolversFactory[T]
}

type Option[T any] func(*settings[T])

func New[T any](opts ...Option[T]) *Comfig[T] {
	s := settings[T]{
		extension: "json",
		parse: func(raw []byte) (T, error) {
			var (
				out  T
				zero T
			)
			err := json.Unmarshal(raw, &out)
			if err != nil {
				return zero, err
			}
			return out, nil
		},
		validate: func(config T) error {
			return nil
		},
		resolversFactories: []ResolversFactory[T]{},
	}

	for _, opt := range opts {
		opt(&s)
	}

	return &Comfig[T]{settings: s}
}

func WithPath[T any](dir string) Option[T] {
	return WithSource[T](NewFileSystemSourceByDirectory(dir))
}

func WithFS[T any](fs fs.FS) Option[T] {
	return WithSource[T](NewFileSystemSource(fs))
}

func WithSource[T any](source Source) Option[T] {
	return func(s *settings[T]) { s.source = source }
}

func WithParser[T any](extension string, parse func(raw []byte) (T, error)) Option[T] {
	return func(s *settings[T]) {
		s.extension = extension
		s.parse = parse
	}
}

func WithValidator[T any](validate func(config T) error) Option[T] {
	return func(s *settings[T]) {
		s.validate = validate
	}
}

func WithResolvers[T any](factory ResolversFactory[T]) Option[T] {
	return func(s *settings[T]) {
		s.resolversFactories = append(s.resolversFactories, factory)
	}
}

type Comfig[T any] struct {
	settings[T]
}

func (l *Comfig[T]) Load(ctx context.Context) (T, error) {
	var zero T

	if l.source == nil {
		return zero, errors.New("no source configured: use WithPath/WithFS or WithSource to add a source")
	}

	if l.extension == "" {
		return zero, errors.New("no file extension set: use WithParser to set a file extension to use")
	}

	if l.validate == nil {
		return zero, errors.New("no validation set: use WithValidator to use a validation function")
	}

	if l.parse == nil {
		return zero, errors.New("no parser set: use WithParser to set a parser to use")
	}

	if l.resolversFactories == nil {
		return zero, errors.New("no resolver factory set: use WithResolvers to add resolvers")
	}

	raw, err := l.source.Configuration(ctx, Environment(), l.extension)
	if err != nil {
		return zero, fmt.Errorf("read configuration: %w", err)
	}

	config, err := l.parse(raw)
	if err != nil {
		return zero, fmt.Errorf("parse configuration: %w", err)
	}

	resolvers, err := buildResolvers(ctx, l.resolversFactories, config)
	if err != nil {
		return zero, fmt.Errorf("build resolvers: %w", err)
	}

	err = resolve(ctx, &config, resolvers)
	if err != nil {
		return zero, fmt.Errorf("resolve values: %w", err)
	}

	if err = l.validate(config); err != nil {
		return zero, fmt.Errorf("validate configuration: %w", err)
	}

	return config, nil
}

func buildResolvers[T any](ctx context.Context, factories []ResolversFactory[T], config T) (map[string]Resolver, error) {
	createdResolvers := []Resolver{}

	for _, factory := range factories {
		resolvers, err := factory(ctx, config)

		if err != nil {
			return map[string]Resolver{}, fmt.Errorf("create resolver(s): %w", err)
		}

		createdResolvers = append(createdResolvers, resolvers...)
	}

	resolvers := map[string]Resolver{}
	var duplicates []string
	for _, resolver := range createdResolvers {
		prefix := resolver.Prefix()
		if _, exists := resolvers[prefix]; exists {
			duplicates = append(duplicates, prefix+"://")
			continue
		}
		resolvers[prefix] = resolver
	}

	if len(duplicates) > 0 {
		return map[string]Resolver{}, fmt.Errorf("duplicate resolver prefixes found, please override prefixes or remove duplicates: %s", strings.Join(duplicates, ", "))
	}

	return resolvers, nil
}
