# comfig-gcp

Resolve Google Cloud Secret Manager values in
[`github.com/philipjohanszon/comfig/go/comfig`](../comfig/README.md) configuration files. For AWS,
see the [`comfig-aws`](../comfig-aws/README.md) adapter.

```bash
go get github.com/philipjohanszon/comfig/go/comfig \
  github.com/philipjohanszon/comfig/go/comfig-gcp
```

## Usage

Create `config/local.json`:

```json
{
  "gcpProjectID": "my-project",
  "token": "gcp://development-api-token"
}
```

Create the resolver from the configuration-defined Google Cloud project ID. Resolver factories
receive the parsed configuration before references are resolved. Retain the resolver so it can be
closed after loading:

```go
package main

import (
	"context"

	"github.com/philipjohanszon/comfig/go/comfig"
	comfiggcp "github.com/philipjohanszon/comfig/go/comfig-gcp"
)

type Config struct {
	GCPProjectID string `json:"gcpProjectID"`
	Token        string `json:"token"`
}

func loadConfig(ctx context.Context) (config Config, err error) {
	var resolver comfiggcp.Resolver

	loader := comfig.New[Config](
		comfig.WithResolvers(func(ctx context.Context, raw Config) ([]comfig.Resolver, error) {
			resolver, err = comfiggcp.NewResolver(ctx, raw.GCPProjectID)
			if err != nil {
				return nil, err
			}

			return []comfig.Resolver{resolver}, nil
		}),
	)

	config, err = loader.Load(ctx)
	if resolver != nil {
		if closeErr := resolver.Close(); err == nil {
			err = closeErr
		}
	}
	return config, err
}
```

`NewResolver` creates a Google Cloud Secret Manager client using the default credential chain.
`Close` must be called after loading; it forwards to the underlying client, including one supplied
with `WithClient`.

## Reference syntax

- `gcp://production-database` uses the `latest` version.
- `gcp://production-database@12` selects version `12`.

Pass `WithClient` to use an existing `*secretmanager.Client`, and `WithPrefixOverride` to register
the resolver under a prefix other than `gcp`.

## Multiple Google Cloud projects

Register a second resolver with a different prefix:

```go
resolver, err := comfiggcp.NewResolver(
	ctx,
	"operations-project",
	comfiggcp.WithPrefixOverride("gcp-ops"),
)
```

It handles references such as `gcp-ops://shared-api-token`.
