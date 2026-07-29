package comfig

import (
	"context"
	"errors"
	"reflect"
	"strings"
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

func TestResolveExpandable(t *testing.T) {
	fakeFileResolver := fakeResolver{
		prefix: "file",
		input: map[string]string{
			"db":     `{"host":"prodserver.com","port":4213,"password":"pw"}`,
			"nested": `{"host":"prodserver.com","password":"env://pw"}`,
		},
	}

	fakeEnvResolver := fakeResolver{
		prefix: "env",
		input: map[string]string{
			"pw":  "resolved-password",
			"sib": "resolved-sibling",
		},
	}

	resolvers := map[string]Resolver{
		fakeFileResolver.Prefix(): fakeFileResolver,
		fakeEnvResolver.Prefix():  fakeEnvResolver,
	}

	t.Run("expands an expandable struct field", func(t *testing.T) {
		conf := struct{ Database Expandable[database] }{
			Database: NewReference[database]("file://db"),
		}

		if err := resolve(context.Background(), &conf, resolvers); err != nil {
			t.Fatalf("failed to resolve: %s", err)
		}

		expected := database{Host: "prodserver.com", Port: 4213, Password: "pw"}
		if !reflect.DeepEqual(conf.Database.Value, expected) {
			t.Fatalf("got=%+v, expected=%+v", conf.Database.Value, expected)
		}
	})

	t.Run("resolves references nested inside an inline expandable value", func(t *testing.T) {
		conf := struct{ Database Expandable[database] }{
			Database: NewExpanded(database{Host: "localhost", Password: "env://pw"}),
		}

		if err := resolve(context.Background(), &conf, resolvers); err != nil {
			t.Fatalf("failed to resolve: %s", err)
		}

		if conf.Database.Value.Password != "resolved-password" {
			t.Fatalf("got=%s, expected=resolved-password", conf.Database.Value.Password)
		}
	})

	t.Run("does not resolve references inside a value expanded from a string", func(t *testing.T) {
		conf := struct{ Database Expandable[database] }{
			Database: NewReference[database]("file://nested"),
		}

		if err := resolve(context.Background(), &conf, resolvers); err != nil {
			t.Fatalf("failed to resolve: %s", err)
		}

		if conf.Database.Value.Password != "env://pw" {
			t.Fatalf("got=%s, expected=env://pw", conf.Database.Value.Password)
		}
	})

	t.Run("does not resolve references inside an expanded value reached a second time", func(t *testing.T) {
		shared := NewReference[database]("file://nested")
		conf := struct {
			First  *Expandable[database]
			Second *Expandable[database]
		}{First: &shared, Second: &shared}

		if err := resolve(context.Background(), &conf, resolvers); err != nil {
			t.Fatalf("failed to resolve: %s", err)
		}

		if conf.First.Value.Password != "env://pw" {
			t.Fatalf("got=%s, expected=env://pw", conf.First.Value.Password)
		}
	})

	t.Run("expands an expandable in a slice element", func(t *testing.T) {
		conf := struct{ Databases []Expandable[database] }{
			Databases: []Expandable[database]{NewReference[database]("file://db")},
		}

		if err := resolve(context.Background(), &conf, resolvers); err != nil {
			t.Fatalf("failed to resolve: %s", err)
		}

		if conf.Databases[0].Value.Host != "prodserver.com" {
			t.Fatalf("got=%+v, expected host=prodserver.com", conf.Databases[0].Value)
		}
	})

	t.Run("expands an expandable stored as a map value", func(t *testing.T) {
		conf := struct {
			Databases map[string]Expandable[database]
		}{
			Databases: map[string]Expandable[database]{"primary": NewReference[database]("file://db")},
		}

		if err := resolve(context.Background(), &conf, resolvers); err != nil {
			t.Fatalf("failed to resolve: %s", err)
		}

		if conf.Databases["primary"].Value.Host != "prodserver.com" {
			t.Fatalf("got=%+v, expected host=prodserver.com", conf.Databases["primary"].Value)
		}
	})

	t.Run("expands an expandable behind a pointer field", func(t *testing.T) {
		reference := NewReference[database]("file://db")
		conf := struct{ Database *Expandable[database] }{Database: &reference}

		if err := resolve(context.Background(), &conf, resolvers); err != nil {
			t.Fatalf("failed to resolve: %s", err)
		}

		if conf.Database.Value.Host != "prodserver.com" {
			t.Fatalf("got=%+v, expected host=prodserver.com", conf.Database.Value)
		}
	})

	t.Run("leaves a nil pointer to an expandable untouched", func(t *testing.T) {
		conf := struct{ Database *Expandable[database] }{}

		if err := resolve(context.Background(), &conf, resolvers); err != nil {
			t.Fatalf("failed to resolve: %s", err)
		}

		if conf.Database != nil {
			t.Fatalf("nil pointer was populated, got=%+v", conf.Database)
		}
	})

	t.Run("expands an expandable held in an any field", func(t *testing.T) {
		conf := struct{ Random any }{Random: NewReference[database]("file://db")}

		if err := resolve(context.Background(), &conf, resolvers); err != nil {
			t.Fatalf("failed to resolve: %s", err)
		}

		expanded, ok := conf.Random.(Expandable[database])
		if !ok {
			t.Fatalf("interface field no longer holds an expandable, got=%T", conf.Random)
		}
		if expanded.Value.Host != "prodserver.com" {
			t.Fatalf("got=%+v, expected host=prodserver.com", expanded.Value)
		}
	})

	t.Run("expands an expandable embedded in a user-defined type", func(t *testing.T) {
		type embedder struct {
			Expandable[database]
			Sibling string
		}

		conf := struct{ Nested embedder }{
			Nested: embedder{Expandable: NewReference[database]("file://db")},
		}

		if err := resolve(context.Background(), &conf, resolvers); err != nil {
			t.Fatalf("failed to resolve: %s", err)
		}

		if conf.Nested.Value.Host != "prodserver.com" {
			t.Fatalf("got=%+v, expected host=prodserver.com", conf.Nested.Value)
		}
	})

	t.Run("resolves the sibling fields of a struct that embeds an expandable", func(t *testing.T) {
		type embedder struct {
			Expandable[database]
			Sibling string
		}

		conf := struct{ Nested embedder }{
			Nested: embedder{
				Expandable: NewReference[database]("file://db"),
				Sibling:    "env://sib",
			},
		}

		if err := resolve(context.Background(), &conf, resolvers); err != nil {
			t.Fatalf("failed to resolve: %s", err)
		}

		if conf.Nested.Sibling != "resolved-sibling" {
			t.Fatalf("got=%s, expected=resolved-sibling", conf.Nested.Sibling)
		}
	})

	t.Run("names the failing field when expansion fails", func(t *testing.T) {
		conf := struct{ Database Expandable[database] }{
			Database: NewReference[database]("file://missing"),
		}

		err := resolve(context.Background(), &conf, resolvers)
		if err == nil {
			t.Fatalf("expected an error for an unresolvable reference")
		}

		if !strings.Contains(err.Error(), "Database") {
			t.Fatalf("error does not name the failing field, got=%s", err)
		}
	})

	t.Run("leaves expandables in unexported fields untouched", func(t *testing.T) {
		type config struct {
			secret Expandable[database]
		}

		conf := config{secret: NewReference[database]("file://db")}

		if err := resolve(context.Background(), &conf, resolvers); err != nil {
			t.Fatalf("failed to resolve: %s", err)
		}

		if !reflect.DeepEqual(conf.secret, NewReference[database]("file://db")) {
			t.Fatalf("unexported field was modified, got=%#v", conf.secret)
		}
	})
}
