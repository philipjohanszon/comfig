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
	resolverSettings := resolverSettings{
		prefix: "env",
	}

	for _, opt := range opts {
		opt(&resolverSettings)
	}

	return envResolver{resolverSettings: resolverSettings}
}

type envResolver struct {
	resolverSettings
}

func (e envResolver) Prefix() string { return e.prefix }
func (e envResolver) Resolve(_ context.Context, value string) (string, error) {
	data, ok := os.LookupEnv(value)
	if !ok {
		return "", fmt.Errorf("environment variable %s is not set", value)
	}
	return data, nil
}

func NewFileResolver(opts ...ResolverOption) Resolver {
	resolverSettings := resolverSettings{
		prefix: "file",
	}

	for _, opt := range opts {
		opt(&resolverSettings)
	}

	return fileResolver{resolverSettings: resolverSettings}
}

type fileResolver struct {
	resolverSettings
}

func (f fileResolver) Prefix() string { return f.prefix }
func (f fileResolver) Resolve(_ context.Context, value string) (string, error) {
	data, err := os.ReadFile(value)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", value, err)
	}
	return string(data), nil
}
