package comfig_gcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
)

type testSecretManagerClient struct {
	request  *secretmanagerpb.AccessSecretVersionRequest
	response *secretmanagerpb.AccessSecretVersionResponse
	err      error
}

func (c *testSecretManagerClient) AccessSecretVersion(_ context.Context, request *secretmanagerpb.AccessSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	c.request = request
	return c.response, c.err
}

type closableTestSecretManagerClient struct {
	*testSecretManagerClient
	closed bool
}

func (c *closableTestSecretManagerClient) Close() error {
	c.closed = true
	return nil
}

func assertGCPRequestName(t *testing.T, request *secretmanagerpb.AccessSecretVersionRequest, name string) {
	t.Helper()

	if request == nil {
		t.Fatal("expected provider request")
	}
	if got := request.GetName(); got != name {
		t.Fatalf("resource name = %q, want %q", got, name)
	}
}

func secretResponse(value string) *secretmanagerpb.AccessSecretVersionResponse {
	return &secretmanagerpb.AccessSecretVersionResponse{
		Payload: &secretmanagerpb.SecretPayload{Data: []byte(value)},
	}
}

func newTestResolver(projectID string, client SecretManagerClient, opts ...ResolverOption) Resolver {
	settings := resolverSettings{
		prefix:    "gcp",
		projectID: projectID,
		client:    client,
	}
	for _, opt := range opts {
		opt(&settings)
	}
	return &resolver{settings: settings}
}

func TestGCPSecretResolver(t *testing.T) {
	t.Run("rejects an empty project ID", func(t *testing.T) {
		_, err := NewResolver(context.Background(), "")
		if err == nil || !strings.Contains(err.Error(), "invalid project id") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("returns the latest secret version for a bare reference", func(t *testing.T) {
		client := &testSecretManagerClient{response: secretResponse("s3cr3t")}
		resolver := newTestResolver("my-proj", client)

		got, err := resolver.Resolve(context.Background(), "db")
		if err != nil {
			t.Fatal(err)
		}
		if got != "s3cr3t" {
			t.Fatalf("got %q", got)
		}
		assertGCPRequestName(t, client.request, "projects/my-proj/secrets/db/versions/latest")
	})

	t.Run("requests an explicitly selected version", func(t *testing.T) {
		client := &testSecretManagerClient{response: secretResponse("old")}
		resolver := newTestResolver("my-proj", client)

		if _, err := resolver.Resolve(context.Background(), "db@3"); err != nil {
			t.Fatal(err)
		}
		assertGCPRequestName(t, client.request, "projects/my-proj/secrets/db/versions/3")
	})

	t.Run("rejects an empty version selector before requesting the provider", func(t *testing.T) {
		client := &testSecretManagerClient{}
		resolver := newTestResolver("my-proj", client)

		if _, err := resolver.Resolve(context.Background(), "db@"); err == nil {
			t.Fatal("expected error")
		}
		if client.request != nil {
			t.Fatal("expected no provider request")
		}
	})

	t.Run("rejects multiple version selectors before requesting the provider", func(t *testing.T) {
		client := &testSecretManagerClient{}
		resolver := newTestResolver("my-proj", client)

		if _, err := resolver.Resolve(context.Background(), "db@3@typo"); err == nil {
			t.Fatal("expected error")
		}
		if client.request != nil {
			t.Fatal("expected no provider request")
		}
	})

	t.Run("returns an error when the secret has no payload", func(t *testing.T) {
		resolver := newTestResolver("my-proj", &testSecretManagerClient{
			response: &secretmanagerpb.AccessSecretVersionResponse{},
		})

		_, err := resolver.Resolve(context.Background(), "db")
		if err == nil || !strings.Contains(err.Error(), "has no payload") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("wraps provider errors", func(t *testing.T) {
		providerErr := errors.New("permission denied")
		resolver := newTestResolver("my-proj", &testSecretManagerClient{err: providerErr})

		_, err := resolver.Resolve(context.Background(), "db")
		if !errors.Is(err, providerErr) {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestGCPSecretResolverOptions(t *testing.T) {
	t.Run("prefix override option changes the prefix which the resolver uses", func(t *testing.T) {
		resolver := newTestResolver("my-proj", &testSecretManagerClient{}, WithPrefixOverride("gcp-production"))
		if got := resolver.Prefix(); got != "gcp-production" {
			t.Fatalf("prefix = %q", got)
		}
	})
}

func TestGCPSecretResolverClose(t *testing.T) {
	t.Run("does not close a client without a close operation", func(t *testing.T) {
		client := &testSecretManagerClient{response: secretResponse("s3cr3t")}
		resolver := newTestResolver("my-proj", client)

		if err := resolver.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := resolver.Resolve(context.Background(), "db"); err != nil {
			t.Fatalf("client was unusable after Close: %v", err)
		}
	})

	t.Run("closes a client that implements io.Closer", func(t *testing.T) {
		client := &closableTestSecretManagerClient{
			testSecretManagerClient: &testSecretManagerClient{},
		}
		resolver := newTestResolver("my-proj", client)

		if err := resolver.Close(); err != nil {
			t.Fatal(err)
		}
		if !client.closed {
			t.Fatal("expected resolver to close its client")
		}
	})
}
