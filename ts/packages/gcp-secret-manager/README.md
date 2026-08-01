# @comfig/gcp-secret-manager

Resolve Google Cloud Secret Manager values in [`@comfig/core`](../core/README.md) configuration
files. For AWS, see the [`@comfig/aws-secrets-manager`](../aws-secrets-manager/README.md) adapter.

```bash
npm install @comfig/core @comfig/gcp-secret-manager zod
```

## Usage

Create `config/local.json`:

```json
{
  "projectId": "my-project",
  "token": "gcp://development-api-token"
}
```

Register the resolver before loading the configuration. Resolver factories receive the parsed
configuration, so the project ID can come from that file:

```ts
import { Comfig } from "@comfig/core"
import { GCPSecretResolver } from "@comfig/gcp-secret-manager"
import { z } from "zod"

const comfig = new Comfig(z.object({
  projectId: z.string(),
  token: z.string(),
}))
  .useResolver((raw) => GCPSecretResolver({ projectId: raw.projectId }))

const config = await comfig.load()
```

`GCPSecretResolver` creates a Google Cloud client using the default credential chain. Supply an
existing `SecretManagerServiceClient` as `client` when your application manages the client itself.

## Reference syntax

- `gcp://production-database` resolves the `latest` version.
- `gcp://production-database@12` resolves version `12`.

## Multiple Google Cloud projects

The optional second argument changes the resolver's prefix, allowing multiple project resolvers in
one configuration:

```ts
comfig.useResolver((raw) => GCPSecretResolver({ projectId: raw.projectId }))
comfig.useResolver(() => GCPSecretResolver({ projectId: "operations-project" }, "gcp-ops"))
```

The second resolver handles references such as `gcp-ops://shared-api-token`.
