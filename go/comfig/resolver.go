package comfig

import (
	"context"
	"fmt"
	"os"
)

type Resolver interface {
	Prefix() string
	Resolve(ctx context.Context, value string) (string, error)
}

type ResolversFactory[T any] func(ctx context.Context, config T) ([]Resolver, error)

type resolverSettings struct {
	prefix string
}

type ResolverOption func(settings *resolverSettings)

func WithPrefixOverride(prefix string) ResolverOption {
	return func(settings *resolverSettings) {
		settings.prefix = prefix
	}
}

func NewEnvResolver(opts ...ResolverOption) Resolver {
	settings := resolverSettings{
		prefix: "env",
	}

	for _, opt := range opts {
		opt(&settings)
	}

	return envResolver{settings: settings}
}

type envResolver struct {
	settings resolverSettings
}

func (e envResolver) Prefix() string { return e.settings.prefix }
func (e envResolver) Resolve(_ context.Context, value string) (string, error) {
	data, ok := os.LookupEnv(value)
	if !ok {
		return "", fmt.Errorf("environment variable %s is not set", value)
	}
	return data, nil
}

func NewFileResolver(opts ...ResolverOption) Resolver {
	settings := resolverSettings{
		prefix: "file",
	}

	for _, opt := range opts {
		opt(&settings)
	}

	return fileResolver{settings: settings}
}

type fileResolver struct {
	settings resolverSettings
}

func (f fileResolver) Prefix() string { return f.settings.prefix }
func (f fileResolver) Resolve(_ context.Context, value string) (string, error) {
	data, err := os.ReadFile(value)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", value, err)
	}
	return string(data), nil
}
