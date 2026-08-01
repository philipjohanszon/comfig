# @comfig/core

Load an environment-specific configuration file, resolve references, and validate the final result
with Zod 4.

```bash
npm install @comfig/core zod
```

By default, `Comfig` reads `config/<environment>.json`, where the environment is the value of the
`env` environment variable, falling back to `ENV`, then to `local`.

```ts
import { Comfig, EnvResolver, expandable } from "@comfig/core"
import { z } from "zod"

const config = await new Comfig(z.object({
  database: expandable(z.object({
    host: z.string(),
    port: z.int(),
  })),
  token: z.string(),
}))
  .useResolver(() => EnvResolver())
  .load()
```

For this `config/local.json`:

```json
{
  "database": { "host": "localhost", "port": 5432 },
  "token": "env://LOCAL_TOKEN"
}
```

`config` is inferred from the Zod schema. `expandable(schema)` accepts either an inline value or a
string resolved by a registered resolver; resolved strings are decoded as JSON by default.

## Configuration

- `usePath(path)` reads `<environment>.<extension>` from another directory.
- `useSource(source)` replaces file loading. A `Source` implements
  `getConfiguration(environment, extension)`.
- `useParser(parser)` changes the parser and file extension.
- `useResolver(factory)` registers a resolver. Factories receive the parsed configuration and may
  return a resolver asynchronously.

`EnvResolver()` handles `env://NAME`; `FileResolver()` handles `file://path`. Missing environment
variables and duplicate resolver prefixes cause an error. Unregistered prefixes remain unchanged.

## Expandable values

Use `expandable` when a field can be inline in one environment and resolved from a secret in
another:

```ts
const schema = z.object({
  database: expandable(z.object({
    host: z.string(),
    port: z.int(),
  })),
})
```

```json
{ "database": { "host": "localhost", "port": 5432 } }
```

```json
{ "database": "aws://production/database" }
```

The resolved string is parsed with `JSON.parse` by default. Pass a parser as the second argument
to support a different representation:

```ts
const tags = expandable(
  z.array(z.string()),
  (raw) => raw.split(",").map((tag) => tag.trim()),
)
```

## Custom sources, parsers, and resolvers

Replace filesystem loading with a `Source`, or change the file extension and parser with a
`Parser`:

```ts
import type { Parser, Source } from "@comfig/core"

const source: Source = {
  getConfiguration: async (environment, extension) =>
    fetchConfiguration(`${environment}.${extension}`),
}

const parser: Parser = {
  extension: "yaml",
  parse: (raw) => parseYaml(raw) as Record<string, unknown>,
}

const comfig = new Comfig(schema)
  .useSource(source)
  .useParser(parser)
```

A resolver declares the prefix it owns and returns the replacement string:

```ts
import type { Resolver } from "@comfig/core"

const vaultResolver: Resolver = {
  prefix: "vault",
  resolve: async (name) => readSecret(name),
}

comfig.useResolver(() => vaultResolver)
```

## Cloud secret managers

Resolve AWS Secrets Manager and Google Cloud Secret Manager references with the adapter packages:

- [`@comfig/aws-secrets-manager`](../aws-secrets-manager/README.md) — `aws://` references
- [`@comfig/gcp-secret-manager`](../gcp-secret-manager/README.md) — `gcp://` references

Install only the adapters your application uses:

```bash
npm install @comfig/aws-secrets-manager @comfig/gcp-secret-manager
```

See the [repository guide](../../../README.md) for the full reference scheme and the Go modules.
