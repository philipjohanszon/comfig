# Comfig

Typed, environment-aware configuration loading for TypeScript and Go.

Comfig loads a configuration file, resolves references to environment variables, files, and cloud
secret managers, then returns a typed value in your application's native types. TypeScript
validates with Zod; Go supports validation through `WithValidator`.

## Features

- **Typed results** — inferred from a Zod schema in TypeScript, from type parameters in Go.
- **Environment-aware** — reads `config/<environment>.json`, where the environment comes from the
  `env` or `ENV` environment variable, falling back to `local`.
- **References** — `env://` and `file://` resolvers included, `aws://` and `gcp://` via adapters.
- **Pluggable** — swap the source, the parser, or add custom resolvers.
- **Expandable fields** — inline values in development, secret references in production, with no
  code changes.
- **Lightweight** — the Go core has no third-party dependencies; the TypeScript core only depends
  on Zod.

## Quick start

TypeScript:

```ts
import { Comfig, EnvResolver } from "@comfig/core"
import { z } from "zod"

const config = await new Comfig(z.object({
  token: z.string(),
}))
  .useResolver(() => EnvResolver())
  .load()
```

Go, where `Config` is a struct with `json` field tags:

```go
func loadConfig(ctx context.Context) (Config, error) {
	return comfig.New[Config](
		comfig.WithResolvers(func(context.Context, Config) ([]comfig.Resolver, error) {
			return []comfig.Resolver{comfig.NewEnvResolver()}, nil
		}),
	).Load(ctx)
}
```

Install `@comfig/core` with `npm install @comfig/core zod`, or
`github.com/philipjohanszon/comfig/go/comfig` (Go 1.26+). Both read `config/<environment>.json`,
where the environment is the value of the `env` environment variable, falling back to `ENV`, then
`local`.

For the full API, expandable values, and custom sources and resolvers, see the
[TypeScript guide](ts/packages/core/README.md) and the [Go guide](go/comfig/README.md).

## Packages

TypeScript:

- [`@comfig/core`](ts/packages/core/README.md) — configuration loading, validation, and built-in resolvers
- [`@comfig/aws-secrets-manager`](ts/packages/aws-secrets-manager/README.md) — AWS Secrets Manager resolver
- [`@comfig/gcp-secret-manager`](ts/packages/gcp-secret-manager/README.md) — Google Cloud Secret Manager resolver

Go:

- [`github.com/philipjohanszon/comfig/go/comfig`](go/comfig/README.md) — configuration loading and built-in resolvers
- [`github.com/philipjohanszon/comfig/go/comfig-aws`](go/comfig-aws/README.md) — AWS Secrets Manager resolver
- [`github.com/philipjohanszon/comfig/go/comfig-gcp`](go/comfig-gcp/README.md) — Google Cloud Secret Manager resolver

## Reference schemes

- `env://NAME` reads the environment variable `NAME`.
- `file://path` reads the contents of `path`.
- `aws://secret`, `aws://secret@stage`, and `aws://secret#version-id` read AWS Secrets Manager.
- `gcp://secret` and `gcp://secret@version` read Google Cloud Secret Manager; the default version
  is `latest`.

A reference with an unregistered prefix is left unchanged, so you only install the resolvers you
use. Missing environment variables and duplicate resolver prefixes return errors. For `aws://`
references, encode a literal `@` in the secret ID as `%40` when selecting a version.

## Cloud secret managers

The adapters use their SDK's default credential chain unless you provide an existing client. Each
adapter has a self-contained guide:

- [`@comfig/aws-secrets-manager`](ts/packages/aws-secrets-manager/README.md) and
  [`comfig-aws`](go/comfig-aws/README.md) for `aws://` references
- [`@comfig/gcp-secret-manager`](ts/packages/gcp-secret-manager/README.md) and
  [`comfig-gcp`](go/comfig-gcp/README.md) for `gcp://` references

## Development

The TypeScript packages use pnpm:

```bash
cd ts && pnpm install && pnpm build && pnpm test
```

Each Go module has its own test suite:

```bash
cd go/comfig && go test ./...
cd ../comfig-aws && go test ./...
cd ../comfig-gcp && go test ./...
```

## License

MIT
