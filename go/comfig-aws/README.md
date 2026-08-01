# comfig-aws

Resolve AWS Secrets Manager values in
[`github.com/philipjohanszon/comfig/go/comfig`](../comfig/README.md) configuration files. For
Google Cloud, see the [`comfig-gcp`](../comfig-gcp/README.md) adapter.

```bash
go get github.com/philipjohanszon/comfig/go/comfig \
  github.com/philipjohanszon/comfig/go/comfig-aws
```

## Usage

Create `config/local.json`:

```json
{
  "awsRegion": "eu-north-1",
  "token": "aws://development/api-token"
}
```

Create the resolver from the configuration-defined AWS region. Resolver factories receive the
parsed configuration before references are resolved:

```go
package main

import (
	"context"

	"github.com/philipjohanszon/comfig/go/comfig"
	"github.com/philipjohanszon/comfig/go/comfig-aws"
)

type Config struct {
	AWSRegion string `json:"awsRegion"`
	Token     string `json:"token"`
}

func loadConfig(ctx context.Context) (Config, error) {
	return comfig.New[Config](
		comfig.WithResolvers(func(ctx context.Context, raw Config) ([]comfig.Resolver, error) {
			resolver, err := comfig_aws.NewResolver(ctx, raw.AWSRegion)
			if err != nil {
				return nil, err
			}

			return []comfig.Resolver{resolver}, nil
		}),
	).Load(ctx)
}
```

`NewResolver` creates an AWS SDK client using the default credential chain. Use
`NewResolverWithClient` to provide an existing `*secretsmanager.Client`.

## Reference syntax

- `aws://production/database` uses AWS's default version selection.
- `aws://production/database@AWSPREVIOUS` selects a `VersionStage`.
- `aws://production/database#version-id` selects a `VersionId`.

When a secret ID contains `@` and you use a selector, encode the `@` as `%40`.

## Multiple AWS accounts or regions

Pass `WithPrefixOverride` to register a second resolver:

```go
resolver, err := comfig_aws.NewResolver(
	ctx,
	"us-east-1",
	comfig_aws.WithPrefixOverride("aws-us"),
)
```

It handles references such as `aws-us://production/api-token`.
