package comfig

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeResolver struct {
	prefix string
	input  map[string]string
}

func (f fakeResolver) Prefix() string { return f.prefix }
func (f fakeResolver) Resolve(_ context.Context, value string) (string, error) {
	output, exists := f.input[value]
	if !exists {
		return "", errors.New("no such value")
	}

	return output, nil
}

func TestResolve(t *testing.T) {
	t.Run("resolves dependencies in configuration", func(t *testing.T) {
		type Token struct {
			For        string
			Token      string
			Additional []string
		}

		type Configuration struct {
			Secret          string
			TokensPerTenant map[string][]Token
			Random          any
			Number          int
			URL             string
		}

		conf := Configuration{
			Secret: "file://secrets/secrets-file",
			TokensPerTenant: map[string][]Token{
				"tenant1": {
					{
						For:        "API1",
						Token:      "gcp://tenant1-api1-token",
						Additional: []string{"file://tenants/1/certificate", "tenant-1"},
					},
					{
						For:        "API2",
						Token:      "gcp://tenant1-api2-token",
						Additional: []string{"file://tenants/1/certificate"},
					},
				},
				"tenant2": {
					{
						For:        "API1",
						Token:      "gcp://tenant2-api1-token",
						Additional: []string{"file://tenants/2/certificate", "tenant-2"},
					},
				},
			},
			Random: []string{"env://random-env1", "env://random-env-2", "file://random-file-1.json"},
			Number: 42,
			URL:    "https://example.com",
		}

		expected := Configuration{
			Secret: "secret-here",
			TokensPerTenant: map[string][]Token{
				"tenant1": {
					{
						For:        "API1",
						Token:      "tenant1-token-for-api1",
						Additional: []string{"tenant1-certificate", "tenant-1"},
					},
					{
						For:        "API2",
						Token:      "tenant1-token-for-api2",
						Additional: []string{"tenant1-certificate"},
					},
				},
				"tenant2": {
					{
						For:        "API1",
						Token:      "tenant2-token-for-api1",
						Additional: []string{"tenant2-certificate", "tenant-2"},
					},
				},
			},
			Random: []string{"random-value1", "random-value2", "random-stuff"},
			Number: 42,
			URL:    "https://example.com",
		}

		fakeGcpResolver := fakeResolver{
			prefix: "gcp",
			input: map[string]string{
				"tenant1-api1-token": "tenant1-token-for-api1",
				"tenant1-api2-token": "tenant1-token-for-api2",
				"tenant2-api1-token": "tenant2-token-for-api1",
			},
		}

		fakeFileResolver := fakeResolver{
			prefix: "file",
			input: map[string]string{
				"secrets/secrets-file":  "secret-here",
				"tenants/1/certificate": "tenant1-certificate",
				"tenants/2/certificate": "tenant2-certificate",
				"random-file-1.json":    "random-stuff",
			},
		}

		fakeEnvResolver := fakeResolver{
			prefix: "env",
			input: map[string]string{
				"random-env1":  "random-value1",
				"random-env-2": "random-value2",
			},
		}

		err := resolve(context.Background(), &conf, map[string]Resolver{
			fakeGcpResolver.Prefix():  fakeGcpResolver,
			fakeFileResolver.Prefix(): fakeFileResolver,
			fakeEnvResolver.Prefix():  fakeEnvResolver,
		})

		if err != nil {
			t.Fatalf("failed to resolve dependencies: %s", err)
		}

		if !reflect.DeepEqual(conf, expected) {
			t.Fatalf("resolved dependencies do not match expected output, got=%+v, expected=%+v", conf, expected)
		}
	})

	t.Run("only resolves map values", func(t *testing.T) {
		type Configuration struct {
			Tokens map[string]string
		}

		conf := Configuration{Tokens: map[string]string{
			"https://api1.com": "https://fetch-token.com",
			"example":          "https://api1.com",
		}}

		fakeHttpsResolver := fakeResolver{
			prefix: "https",
			input: map[string]string{
				"api1.com":        "api1-response",
				"fetch-token.com": "token",
			},
		}

		err := resolve(context.Background(), &conf, map[string]Resolver{
			fakeHttpsResolver.Prefix(): fakeHttpsResolver,
		})

		if err != nil {
			t.Fatalf("failed to resolve dependencies: %s", err)
		}

		val, exists := conf.Tokens["https://api1.com"]
		if !exists {
			t.Fatalf("no token found for https://api1.com")
		}
		expected := fakeHttpsResolver.input["fetch-token.com"]
		if val != expected {
			t.Fatalf("invalid token found for fetch-token.com, expected=%s, got=%s", expected, val)
		}

		val, exists = conf.Tokens["example"]
		if !exists {
			t.Fatalf("no token found for example")
		}
		expected = fakeHttpsResolver.input["api1.com"]
		if val != expected {
			t.Fatalf("invalid token found for api1.com, expected=%s, got=%s", expected, val)
		}
	})
}
