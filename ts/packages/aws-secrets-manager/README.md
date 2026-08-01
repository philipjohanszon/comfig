# @comfig/aws-secrets-manager

Resolve AWS Secrets Manager values in [`@comfig/core`](../core/README.md) configuration files. For
Google Cloud, see the [`@comfig/gcp-secret-manager`](../gcp-secret-manager/README.md) adapter.

```bash
npm install @comfig/core @comfig/aws-secrets-manager zod
```

## Usage

Create `config/local.json`:

```json
{
  "awsRegion": "eu-north-1",
  "token": "aws://development/api-token"
}
```

Register the resolver before loading the configuration. Resolver factories receive the parsed
configuration, so the AWS region can be defined alongside the reference:

```ts
import { AWSSecretsResolver } from "@comfig/aws-secrets-manager"
import { Comfig } from "@comfig/core"
import { z } from "zod"

const comfig = new Comfig(z.object({
  awsRegion: z.string(),
  token: z.string(),
}))
  .useResolver((raw) => AWSSecretsResolver({ region: raw.awsRegion }))

const config = await comfig.load()
```

`AWSSecretsResolver` creates an AWS SDK client using the default credential chain. `region` is
optional; provide it when your application should use a specific region. Supply an existing
`SecretsManagerClient` as `client` when your application manages the client itself.

## Reference syntax

- `aws://production/database` uses AWS's default version selection.
- `aws://production/database@AWSPREVIOUS` selects a `VersionStage`.
- `aws://production/database#version-id` selects a `VersionId`.

When a secret ID contains `@` and you use a selector, encode the `@` as `%40`.

## Multiple AWS accounts or regions

The optional second argument changes the resolver's prefix, allowing multiple AWS resolvers in one
configuration:

```ts
comfig.useResolver(() => AWSSecretsResolver({ region: "eu-north-1" }))
comfig.useResolver(() => AWSSecretsResolver({ region: "us-east-1" }, "aws-us"))
```

The second resolver handles references such as `aws-us://production/api-token`.
