package comfig_aws

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type testSecretManagerClient struct {
	input    *secretsmanager.GetSecretValueInput
	response *secretsmanager.GetSecretValueOutput
	err      error
}

func (c *testSecretManagerClient) GetSecretValue(_ context.Context, input *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	c.input = input
	return c.response, c.err
}

func assertAWSInput(t *testing.T, input *secretsmanager.GetSecretValueInput, secretID string, versionID, versionStage *string) {
	t.Helper()

	if input == nil {
		t.Fatal("expected provider request")
	}
	if got := aws.ToString(input.SecretId); got != secretID {
		t.Fatalf("secret ID = %q, want %q", got, secretID)
	}
	if got := aws.ToString(input.VersionId); got != aws.ToString(versionID) || (input.VersionId == nil) != (versionID == nil) {
		t.Fatalf("version ID = %v, want %v", input.VersionId, versionID)
	}
	if got := aws.ToString(input.VersionStage); got != aws.ToString(versionStage) || (input.VersionStage == nil) != (versionStage == nil) {
		t.Fatalf("version stage = %v, want %v", input.VersionStage, versionStage)
	}
}

func TestAWSSecretsResolver(t *testing.T) {
	t.Run("returns a string secret", func(t *testing.T) {
		client := &testSecretManagerClient{
			response: &secretsmanager.GetSecretValueOutput{SecretString: aws.String("s3cr3t")},
		}
		resolver := newResolver(client)

		got, err := resolver.Resolve(context.Background(), "prod/db/password")
		if err != nil {
			t.Fatal(err)
		}
		if got != "s3cr3t" {
			t.Fatalf("got %q, want %q", got, "s3cr3t")
		}
	})

	t.Run("requests the current version when reference has no selector", func(t *testing.T) {
		client := &testSecretManagerClient{
			response: &secretsmanager.GetSecretValueOutput{SecretString: aws.String("s3cr3t")},
		}
		resolver := newResolver(client)

		if _, err := resolver.Resolve(context.Background(), "prod/db/password"); err != nil {
			t.Fatal(err)
		}
		assertAWSInput(t, client.input, "prod/db/password", nil, nil)
	})

	t.Run("selects a version stage", func(t *testing.T) {
		client := &testSecretManagerClient{
			response: &secretsmanager.GetSecretValueOutput{SecretString: aws.String("previous")},
		}
		resolver := newResolver(client)

		if _, err := resolver.Resolve(context.Background(), "prod/db/password@AWSPREVIOUS"); err != nil {
			t.Fatal(err)
		}
		assertAWSInput(t, client.input, "prod/db/password", nil, aws.String("AWSPREVIOUS"))
	})

	t.Run("selects a version ID", func(t *testing.T) {
		client := &testSecretManagerClient{
			response: &secretsmanager.GetSecretValueOutput{SecretString: aws.String("pinned")},
		}
		resolver := newResolver(client)

		if _, err := resolver.Resolve(context.Background(), "prod/db/password#version-id"); err != nil {
			t.Fatal(err)
		}
		assertAWSInput(t, client.input, "prod/db/password", aws.String("version-id"), nil)
	})

	t.Run("decodes percent-escaped selectors and secret names", func(t *testing.T) {
		client := &testSecretManagerClient{
			response: &secretsmanager.GetSecretValueOutput{SecretString: aws.String("previous")},
		}
		resolver := newResolver(client)

		if _, err := resolver.Resolve(context.Background(), "prod%40blue/db/password@AWS%50REVIOUS"); err != nil {
			t.Fatal(err)
		}
		assertAWSInput(t, client.input, "prod@blue/db/password", nil, aws.String("AWSPREVIOUS"))
	})

	t.Run("returns binary secrets as UTF-8", func(t *testing.T) {
		client := &testSecretManagerClient{
			response: &secretsmanager.GetSecretValueOutput{SecretBinary: []byte("binary-secret")},
		}
		resolver := newResolver(client)

		got, err := resolver.Resolve(context.Background(), "prod/db/password")
		if err != nil {
			t.Fatal(err)
		}
		if got != "binary-secret" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("returns an error when the secret has no value", func(t *testing.T) {
		resolver := newResolver(&testSecretManagerClient{
			response: &secretsmanager.GetSecretValueOutput{},
		})

		_, err := resolver.Resolve(context.Background(), "prod/db/password")
		if err == nil || !strings.Contains(err.Error(), "has no value") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("wraps provider errors", func(t *testing.T) {
		providerErr := errors.New("access denied")
		resolver := newResolver(&testSecretManagerClient{err: providerErr})

		_, err := resolver.Resolve(context.Background(), "prod/db/password")
		if !errors.Is(err, providerErr) {
			t.Fatalf("error = %v", err)
		}
	})

	for _, reference := range []string{
		"",
		"prod/db/password@",
		"prod/db/password#",
		"prod/db/password#one#two",
		"prod/db/password@one@two",
		"prod@blue/db/password@AWSCURRENT",
		"prod%4/db/password",
	} {
		t.Run("rejects invalid reference "+reference, func(t *testing.T) {
			client := &testSecretManagerClient{}
			resolver := newResolver(client)

			if _, err := resolver.Resolve(context.Background(), reference); err == nil {
				t.Fatal("expected error")
			}
			if client.input != nil {
				t.Fatal("expected no provider request")
			}
		})
	}
}

func TestAWSSecretsResolverPrefixOverride(t *testing.T) {
	resolver := newResolver(&testSecretManagerClient{}, WithPrefixOverride("aws-production"))
	if got := resolver.Prefix(); got != "aws-production" {
		t.Fatalf("prefix = %q", got)
	}
}
