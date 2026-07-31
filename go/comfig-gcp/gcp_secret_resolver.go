package comfig_gcp

import (
	"context"
	"fmt"
	"io"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/philipjohanszon/comfig/go/comfig"
)

type Resolver interface {
	comfig.Resolver
	io.Closer
}

type SecretManagerClient interface {
	AccessSecretVersion(context.Context, *secretmanagerpb.AccessSecretVersionRequest, ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
}

type resolverSettings struct {
	prefix    string
	projectID string
	client    SecretManagerClient
}

type ResolverOption func(*resolverSettings)

func WithPrefixOverride(prefix string) ResolverOption {
	return func(settings *resolverSettings) {
		settings.prefix = prefix
	}
}

func WithClient(client *secretmanager.Client) ResolverOption {
	return func(settings *resolverSettings) {
		settings.client = client
	}
}

func NewResolver(ctx context.Context, projectID string, opts ...ResolverOption) (Resolver, error) {
	if projectID == "" {
		return nil, fmt.Errorf("invalid project id since it was empty")
	}

	settings := resolverSettings{
		projectID: projectID,
		prefix:    "gcp",
		client:    nil,
	}
	for _, opt := range opts {
		opt(&settings)
	}

	if settings.client == nil {
		client, err := secretmanager.NewClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("create GCP Secret Manager client: %w", err)
		}
		settings.client = client
	}

	return &resolver{settings: settings}, nil
}

type resolver struct {
	settings resolverSettings
}

func (r *resolver) Prefix() string {
	return r.settings.prefix
}

func (r *resolver) Resolve(ctx context.Context, value string) (string, error) {
	name, err := resolveName(value, r.settings.projectID)
	if err != nil {
		return "", err
	}

	response, err := r.settings.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return "", fmt.Errorf("access secret version: %w", err)
	}
	if response == nil || response.Payload == nil || response.Payload.Data == nil {
		return "", fmt.Errorf("secret %s has no payload", name)
	}

	return string(response.Payload.Data), nil
}

func (r *resolver) Close() error {
	if closer, ok := r.settings.client.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}

func resolveName(value, projectID string) (string, error) {
	parts := strings.Split(value, "@")
	if len(parts) > 2 {
		return "", fmt.Errorf("GCP secret reference has multiple version selectors")
	}

	secret, version := parts[0], "latest"
	if len(parts) == 2 {
		version = parts[1]
		if version == "" {
			return "", fmt.Errorf("GCP secret reference has an empty version selector")
		}
	}

	return fmt.Sprintf("projects/%s/secrets/%s/versions/%s", projectID, secret, version), nil
}
