package comfig_aws

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/philipjohanszon/comfig/go/comfig"
)

type SecretManagerClient interface {
	GetSecretValue(context.Context, *secretsmanager.GetSecretValueInput, ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type resolverSettings struct {
	prefix string
	client SecretManagerClient
}

type ResolverOption func(*resolverSettings)

func WithPrefixOverride(prefix string) ResolverOption {
	return func(settings *resolverSettings) {
		settings.prefix = prefix
	}
}

func NewResolver(ctx context.Context, region string, opts ...ResolverOption) (comfig.Resolver, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	client := secretsmanager.NewFromConfig(cfg)
	return newResolver(client, opts...), nil
}

func NewResolverWithClient(client *secretsmanager.Client, opts ...ResolverOption) (comfig.Resolver, error) {
	return newResolver(client, opts...), nil
}

type resolver struct {
	settings resolverSettings
}

func newResolver(client SecretManagerClient, opts ...ResolverOption) comfig.Resolver {
	settings := resolverSettings{
		prefix: "aws",
		client: client,
	}
	for _, opt := range opts {
		opt(&settings)
	}

	return &resolver{settings: settings}
}

func (r *resolver) Prefix() string {
	return r.settings.prefix
}

func (r *resolver) Resolve(ctx context.Context, value string) (string, error) {
	reference, err := parseReference(value)
	if err != nil {
		return "", err
	}

	response, err := r.settings.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId:     &reference.secretID,
		VersionId:    reference.versionID,
		VersionStage: reference.versionStage,
	})
	if err != nil {
		return "", fmt.Errorf("get secret value: %w", err)
	}

	if response == nil {
		return "", fmt.Errorf("got unexpected empty response for secret %s", value)
	}

	if response.SecretString != nil {
		return *response.SecretString, nil
	}
	if response.SecretBinary != nil {
		return string(response.SecretBinary), nil
	}

	return "", fmt.Errorf("secret %s has no value", value)
}

type secretReference struct {
	secretID     string
	versionID    *string
	versionStage *string
}

func parseReference(value string) (secretReference, error) {
	if value == "" {
		return secretReference{}, fmt.Errorf("AWS secret can't be empty")
	}

	reference, err := parseVersionIDReference(value)
	if err != nil {
		return secretReference{}, err
	}
	if reference != nil {
		return *reference, nil
	}

	reference, err = parseVersionStageReference(value)
	if err != nil {
		return secretReference{}, err
	}
	if reference != nil {
		return *reference, nil
	}

	secretID, err := decode(value, "secret ID")
	if err != nil {
		return secretReference{}, err
	}
	return secretReference{secretID: secretID}, nil
}

func parseVersionIDReference(value string) (*secretReference, error) {
	parts := strings.Split(value, "#")
	if len(parts) == 1 {
		return nil, nil
	}
	if len(parts) > 2 {
		return nil, fmt.Errorf("AWS secret reference has multiple version ID selectors")
	}

	rawSecretID, rawVersionID := parts[0], parts[1]
	if rawSecretID == "" {
		return nil, fmt.Errorf("AWS secret reference has an empty secret ID")
	}
	if strings.Contains(rawSecretID, "@") {
		return nil, fmt.Errorf("AWS secret IDs containing @ must use %%40 when selecting a version ID")
	}
	if rawVersionID == "" {
		return nil, fmt.Errorf("AWS secret reference has an empty version ID")
	}

	secretID, err := decode(rawSecretID, "secret ID")
	if err != nil {
		return nil, err
	}
	versionID, err := decode(rawVersionID, "version ID")
	if err != nil {
		return nil, err
	}

	return &secretReference{secretID: secretID, versionID: &versionID}, nil
}

func parseVersionStageReference(value string) (*secretReference, error) {
	parts := strings.Split(value, "@")
	if len(parts) == 1 {
		return nil, nil
	}
	if len(parts) > 2 {
		return nil, fmt.Errorf("AWS secret IDs containing @ must use %%40 when selecting a version stage")
	}

	rawSecretID, rawVersionStage := parts[0], parts[1]
	if rawSecretID == "" {
		return nil, fmt.Errorf("AWS secret reference has an empty secret ID")
	}
	if rawVersionStage == "" {
		return nil, fmt.Errorf("AWS secret reference has an empty version stage")
	}

	secretID, err := decode(rawSecretID, "secret ID")
	if err != nil {
		return nil, err
	}
	versionStage, err := decode(rawVersionStage, "version stage")
	if err != nil {
		return nil, err
	}

	return &secretReference{secretID: secretID, versionStage: &versionStage}, nil
}

func decode(value, description string) (string, error) {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return "", fmt.Errorf("AWS secret reference has invalid percent-encoded %s", description)
	}
	return decoded, nil
}
