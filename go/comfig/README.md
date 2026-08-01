# comfig

Load an environment-specific configuration file, resolve references, and return a typed Go value.
Comfig requires Go 1.26 or later.

```bash
go get github.com/philipjohanszon/comfig/go/comfig
```

By default, Comfig reads `config/<environment>.json`, where the environment is the value of the
`env` environment variable, falling back to `ENV`, then to `local`.

```go
package main

import (
	"context"

	"github.com/philipjohanszon/comfig/go/comfig"
)

type Config struct {
	Token string `json:"token"`
}

func loadConfig(ctx context.Context) (Config, error) {
	return comfig.New[Config](
		comfig.WithResolvers(func(context.Context, Config) ([]comfig.Resolver, error) {
			return []comfig.Resolver{comfig.NewEnvResolver()}, nil
		}),
	).Load(ctx)
}
```

With `config/local.json`:

```json
{ "token": "env://LOCAL_TOKEN" }
```

`NewEnvResolver()` resolves `env://NAME`; `NewFileResolver()` resolves `file://path`. Missing
environment variables and duplicate resolver prefixes cause an error. Unregistered prefixes remain
unchanged.

## Options

- `WithPath(dir)` reads configuration from a directory.
- `WithFS(filesystem)` reads from any `fs.FS`, including `embed.FS`.
- `WithSource(source)` replaces file loading. A `Source` implements
  `Configuration(ctx, environment, extension)`.
- `WithParser(extension, parse)` changes parsing and the file extension.
- `WithValidator(validate)` runs after reference resolution.
- `WithResolvers(factory)` registers resolvers. A factory receives the parsed configuration and may
  return multiple resolvers.

Use `Expandable[T]` for a field that accepts an inline value or a string reference. Read its
resolved value from `.Value`; see [Expandable values](#expandable-values) for how references are
expanded.

## Expandable values

Use `Expandable[T]` when development configuration is inline but deployed configuration comes from
a resolver:

```go
type Database struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type Config struct {
	Database comfig.Expandable[Database] `json:"database"`
}
```

```json
{ "database": { "host": "localhost", "port": 5432 } }
```

```json
{ "database": "aws://production/database" }
```

The resolved string expands by calling `Expand(string) error` on `T` when it implements `Expander`,
then `encoding.TextUnmarshaler`, then by assigning it to a string, and finally by decoding JSON.

## Custom sources, parsers, and resolvers

`WithSource` accepts any type that implements `Source`; `WithFS` accepts any `fs.FS`, including a
subdirectory of an embedded filesystem. `WithParser` takes both an extension and a parser:

```go
loader := comfig.New[Config](
	comfig.WithSource[Config](source),
	comfig.WithParser("yaml", parseYAML),
)
```

Bundle configuration files into the binary with `embed.FS`:

```go
//go:embed config
var configFS embed.FS

loader := comfig.New[Config](
	comfig.WithFS(configFS),
	comfig.WithResolvers(func(context.Context, Config) ([]comfig.Resolver, error) {
		return []comfig.Resolver{comfig.NewEnvResolver()}, nil
	}),
)
```

Implement `Resolver` to add a reference scheme:

```go
type vaultResolver struct{}

func (vaultResolver) Prefix() string { return "vault" }

func (vaultResolver) Resolve(ctx context.Context, name string) (string, error) {
	return readSecret(ctx, name)
}
```

Register it from a `WithResolvers` factory. Resolver factories receive the parsed configuration,
which is how a resolver can use configuration-defined settings such as an AWS region, Google Cloud
project ID, or endpoint.

## Cloud secret managers

Resolve AWS Secrets Manager and Google Cloud Secret Manager references with the adapter modules:

- [`comfig-aws`](../comfig-aws/README.md) — `aws://` references
- [`comfig-gcp`](../comfig-gcp/README.md) — `gcp://` references

Install only the adapters your application uses:

```bash
go get github.com/philipjohanszon/comfig/go/comfig-aws
go get github.com/philipjohanszon/comfig/go/comfig-gcp
```

See the [repository guide](../../README.md) for the full reference scheme and the TypeScript
packages.
