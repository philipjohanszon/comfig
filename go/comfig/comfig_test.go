package comfig

import (
	"context"
	"encoding/xml"
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

type memorySource struct {
	data string
	err  error
}

func (m memorySource) Configuration(_ context.Context, _, _ string) ([]byte, error) {
	return []byte(m.data), m.err
}

func TestLoad(t *testing.T) {
	type config struct {
		Name string `json:"name"`
	}

	t.Run("errors when source cannot read configuration", func(t *testing.T) {
		sourceErr := errors.New("source unavailable")
		loader := New[config](WithSource[config](memorySource{err: sourceErr}))

		_, err := loader.Load(context.Background())

		if !errors.Is(err, sourceErr) {
			t.Fatalf("expected source error to be wrapped, got %v", err)
		}
		if !strings.Contains(err.Error(), "read configuration") {
			t.Fatalf("expected read configuration context, got %v", err)
		}
	})

	t.Run("errors when configuration cannot be parsed", func(t *testing.T) {
		loader := New[config](WithSource[config](memorySource{data: `{"name":`}))

		_, err := loader.Load(context.Background())

		if err == nil {
			t.Fatal("expected error for invalid configuration")
		}
		if !strings.Contains(err.Error(), "parse configuration") {
			t.Fatalf("expected parse configuration context, got %v", err)
		}
	})

	t.Run("errors when resolver factory fails", func(t *testing.T) {
		factoryErr := errors.New("credentials unavailable")
		loader := New[config](
			WithSource[config](memorySource{data: `{}`}),
			WithResolvers(func(context.Context, config) ([]Resolver, error) {
				return nil, factoryErr
			}),
		)

		_, err := loader.Load(context.Background())

		if !errors.Is(err, factoryErr) {
			t.Fatalf("expected factory error to be wrapped, got %v", err)
		}
		if !strings.Contains(err.Error(), "build resolvers") {
			t.Fatalf("expected build resolvers context, got %v", err)
		}
	})

	t.Run("errors when resolver prefixes are duplicated", func(t *testing.T) {
		loader := New[config](
			WithSource[config](memorySource{data: `{}`}),
			WithResolvers(func(context.Context, config) ([]Resolver, error) {
				return []Resolver{
					fakeResolver{prefix: "secret"},
					fakeResolver{prefix: "secret"},
				}, nil
			}),
		)

		_, err := loader.Load(context.Background())

		if err == nil {
			t.Fatal("expected error for duplicate resolver prefixes")
		}
		if !strings.Contains(err.Error(), "duplicate resolver prefixes") {
			t.Fatalf("expected duplicate-prefix context, got %v", err)
		}
	})

	t.Run("errors when validation fails", func(t *testing.T) {
		validationErr := errors.New("name is not allowed")
		loader := New[config](
			WithSource[config](memorySource{data: `{"name":"invalid"}`}),
			WithValidator(func(config) error {
				return validationErr
			}),
		)

		_, err := loader.Load(context.Background())

		if !errors.Is(err, validationErr) {
			t.Fatalf("expected validation error to be wrapped, got %v", err)
		}
		if !strings.Contains(err.Error(), "validate configuration") {
			t.Fatalf("expected validation context, got %v", err)
		}
	})

	t.Run("loads the environment configuration that is selected", func(t *testing.T) {
		t.Setenv("env", "dev")

		loader := New[config](WithFS[config](fstest.MapFS{
			"dev.json": &fstest.MapFile{Data: []byte(`{"name":"from-fs"}`)},
		}))

		loaded, err := loader.Load(context.Background())
		if err != nil {
			t.Fatalf("failed to load configuration: %v", err)
		}
		if loaded.Name != "from-fs" {
			t.Fatalf("got name %q, want %q", loaded.Name, "from-fs")
		}
	})
}

func TestLoadExpandable(t *testing.T) {
	type config struct {
		Database Expandable[database]  `json:"database"`
		Tags     Expandable[commaList] `json:"tags"`
	}

	fakeFileResolver := fakeResolver{
		prefix: "file",
		input:  map[string]string{"db": `{"host":"prodserver.com","port":4213,"password":"pw"}`},
	}

	resolvers := func(_ context.Context, _ config) ([]Resolver, error) {
		return []Resolver{fakeFileResolver}, nil
	}

	t.Run("loads a configuration whose expandable field is a reference", func(t *testing.T) {
		loader := New[config](
			WithSource[config](memorySource{data: `{"database":"file://db","tags":"go, config"}`}),
			WithResolvers(resolvers),
		)

		loaded, err := loader.Load(context.Background())
		if err != nil {
			t.Fatalf("failed to load: %s", err)
		}

		expectedDB := database{Host: "prodserver.com", Port: 4213, Password: "pw"}
		if !reflect.DeepEqual(loaded.Database.Value, expectedDB) {
			t.Fatalf("got=%+v, expected=%+v", loaded.Database.Value, expectedDB)
		}

		expectedTags := commaList{"go", "config"}
		if !reflect.DeepEqual(loaded.Tags.Value, expectedTags) {
			t.Fatalf("got=%+v, expected=%+v", loaded.Tags.Value, expectedTags)
		}
	})

	t.Run("errors when an expandable field is absent from the configuration", func(t *testing.T) {
		loader := New[config](
			WithSource[config](memorySource{data: `{"tags":"go, config"}`}),
			WithResolvers(resolvers),
		)

		_, err := loader.Load(context.Background())
		if err == nil {
			t.Fatalf("expected an error for a configuration missing an expandable field")
		}

		if !strings.Contains(err.Error(), "Database") {
			t.Fatalf("error does not name the absent field, got=%s", err)
		}
	})

	t.Run("loads a configuration whose expandable field is written inline", func(t *testing.T) {
		loader := New[config](
			WithSource[config](memorySource{data: `{"database":{"host":"localhost","port":5432},"tags":"go, config"}`}),
			WithResolvers(resolvers),
		)

		loaded, err := loader.Load(context.Background())
		if err != nil {
			t.Fatalf("failed to load: %s", err)
		}

		expectedDB := database{Host: "localhost", Port: 5432}
		if !reflect.DeepEqual(loaded.Database.Value, expectedDB) {
			t.Fatalf("got=%+v, expected=%+v", loaded.Database.Value, expectedDB)
		}

		expectedTags := commaList{"go", "config"}
		if !reflect.DeepEqual(loaded.Tags.Value, expectedTags) {
			t.Fatalf("got=%+v, expected=%+v", loaded.Tags.Value, expectedTags)
		}
	})

}

type xmlExpandable[T any] struct {
	Expandable[T]
}

func (x *xmlExpandable[T]) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var node struct {
		Chardata string `xml:",chardata"`
		Inner    string `xml:",innerxml"`
	}

	if err := d.DecodeElement(&node, &start); err != nil {
		return err
	}

	if !strings.Contains(node.Inner, "<") {
		x.Expandable = NewReference[T](strings.TrimSpace(node.Chardata))
		return nil
	}

	var value T
	if err := xml.Unmarshal([]byte("<value>"+node.Inner+"</value>"), &value); err != nil {
		return err
	}

	x.Expandable = NewExpanded(value)
	return nil
}

func TestLoadExpandableWithCustomParser(t *testing.T) {
	t.Run("loads a configuration whose expandable field is a reference", func(t *testing.T) {
		type config struct {
			Database xmlExpandable[database] `xml:"database"`
		}

		fakeFileResolver := fakeResolver{
			prefix: "file",
			input:  map[string]string{"db": `{"host":"prodserver.com","port":4213,"password":"pw"}`},
		}

		loader := New[config](
			WithSource[config](memorySource{data: `<config><database>file://db</database></config>`}),
			WithParser("xml", func(raw []byte) (config, error) {
				var out config
				return out, xml.Unmarshal(raw, &out)
			}),
			WithResolvers(func(_ context.Context, _ config) ([]Resolver, error) {
				return []Resolver{fakeFileResolver}, nil
			}),
		)

		loaded, err := loader.Load(context.Background())
		if err != nil {
			t.Fatalf("failed to load: %s", err)
		}

		expected := database{Host: "prodserver.com", Port: 4213, Password: "pw"}
		if !reflect.DeepEqual(loaded.Database.Value, expected) {
			t.Fatalf("got=%+v, expected=%+v", loaded.Database.Value, expected)
		}
	})

	t.Run("loads a configuration whose expandable field is written inline", func(t *testing.T) {
		type config struct {
			Database xmlExpandable[database] `xml:"database"`
		}

		fakeEnvResolver := fakeResolver{
			prefix: "env",
			input:  map[string]string{"DB_PASSWORD": "resolved-password"},
		}

		loader := New[config](
			WithSource[config](memorySource{
				data: `<config><database><host>localhost</host><port>5432</port><password>env://DB_PASSWORD</password></database></config>`,
			}),
			WithParser("xml", func(raw []byte) (config, error) {
				var out config
				return out, xml.Unmarshal(raw, &out)
			}),
			WithResolvers(func(_ context.Context, _ config) ([]Resolver, error) {
				return []Resolver{fakeEnvResolver}, nil
			}),
		)

		loaded, err := loader.Load(context.Background())
		if err != nil {
			t.Fatalf("failed to load: %s", err)
		}

		expected := database{Host: "localhost", Port: 5432, Password: "resolved-password"}
		if !reflect.DeepEqual(loaded.Database.Value, expected) {
			t.Fatalf("got=%+v, expected=%+v", loaded.Database.Value, expected)
		}
	})
}
