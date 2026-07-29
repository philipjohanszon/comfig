package comfig

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type database struct {
	Host     string `json:"host" xml:"host"`
	Port     int    `json:"port" xml:"port"`
	Password string `json:"password" xml:"password"`
}

type commaList []string

func (c *commaList) Expand(raw string) error {
	for _, part := range strings.Split(raw, ",") {
		*c = append(*c, strings.TrimSpace(part))
	}
	return nil
}

type textList []string

func (t *textList) UnmarshalText(text []byte) error {
	for _, part := range strings.Split(string(text), ",") {
		*t = append(*t, strings.TrimSpace(part))
	}
	return nil
}

type bothList []string

func (b *bothList) Expand(raw string) error {
	*b = bothList{"expand"}
	return nil
}

func (b *bothList) UnmarshalText(text []byte) error {
	*b = bothList{"text"}
	return nil
}

type name string

func TestExpandableUnmarshalJSON(t *testing.T) {
	t.Run("unmarshals an inline object into Value", func(t *testing.T) {
		var e Expandable[database]

		if err := json.Unmarshal([]byte(`{"host":"localhost","port":5432,"password":"pw"}`), &e); err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		expected := database{Host: "localhost", Port: 5432, Password: "pw"}
		if !reflect.DeepEqual(e.Value, expected) {
			t.Fatalf("inline object was not decoded, got=%+v, expected=%+v", e.Value, expected)
		}
	})

	t.Run("records a JSON string as a deferred reference without decoding it", func(t *testing.T) {
		var e Expandable[database]

		if err := json.Unmarshal([]byte(`"file://./secrets/db"`), &e); err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		if e.isExpanded {
			t.Fatalf("string was not deferred")
		}
		if *e.raw != "file://./secrets/db" {
			t.Fatalf("reference was not recorded, got=%s, expected=file://./secrets/db", *e.raw)
		}
		if !reflect.DeepEqual(e.Value, database{}) {
			t.Fatalf("deferred reference should not decode into Value, got=%+v", e.Value)
		}
	})

	t.Run("leaves the receiver unchanged when the JSON value is null", func(t *testing.T) {
		e := NewExpanded(database{Host: "default"})

		if err := json.Unmarshal([]byte(`null`), &e); err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		if e.Value.Host != "default" {
			t.Fatalf("null overwrote the existing value, got=%+v", e.Value)
		}
	})

	t.Run("clears the previous value when re-unmarshalled as a reference", func(t *testing.T) {
		e := NewExpanded(database{Host: "stale", Port: 1234})

		if err := json.Unmarshal([]byte(`"file://./secrets/db"`), &e); err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		if !reflect.DeepEqual(e.Value, database{}) {
			t.Fatalf("stale value survived, got=%+v", e.Value)
		}
	})

	t.Run("clears the previous reference when re-unmarshalled as an inline value", func(t *testing.T) {
		e := NewReference[database]("file://./secrets/db")

		if err := json.Unmarshal([]byte(`{"host":"localhost"}`), &e); err != nil {
			t.Fatalf("failed to unmarshal: %s", err)
		}

		if !e.isExpanded {
			t.Fatalf("stale isExpanded flag survived")
		}
		if e.raw != nil {
			t.Fatalf("stale reference survived, got=%s", *e.raw)
		}
	})

	t.Run("errors when an inline value does not match the target type", func(t *testing.T) {
		var e Expandable[database]

		if err := json.Unmarshal([]byte(`42`), &e); err == nil {
			t.Fatalf("expected an error for a number decoded into a struct")
		}
	})

	t.Run("leaves Value zero rather than partially populated when an inline object fails to decode", func(t *testing.T) {
		var e Expandable[database]

		if err := json.Unmarshal([]byte(`{"host":"localhost","port":"not-a-number"}`), &e); err == nil {
			t.Fatalf("expected an error for a string decoded into an int field")
		}

		if !reflect.DeepEqual(e.Value, database{}) {
			t.Fatalf("failed decode left partial state, got=%+v", e.Value)
		}
	})
}

func TestExpandString(t *testing.T) {
	t.Run("decodes a JSON document into the target", func(t *testing.T) {
		var got database

		if err := expandString(`{"host":"localhost","port":5432}`, &got); err != nil {
			t.Fatalf("failed to expand: %s", err)
		}

		expected := database{Host: "localhost", Port: 5432}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("got=%+v, expected=%+v", got, expected)
		}
	})

	t.Run("uses Expand when the target implements Expander", func(t *testing.T) {
		var got commaList

		if err := expandString("go, config, expandable", &got); err != nil {
			t.Fatalf("failed to expand: %s", err)
		}

		expected := commaList{"go", "config", "expandable"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("got=%+v, expected=%+v", got, expected)
		}
	})

	t.Run("prefers Expand over UnmarshalText when the target implements both", func(t *testing.T) {
		var got bothList

		if err := expandString("anything", &got); err != nil {
			t.Fatalf("failed to expand: %s", err)
		}

		expected := bothList{"expand"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("got=%+v, expected=%+v", got, expected)
		}
	})

	t.Run("uses UnmarshalText when the target implements encoding.TextUnmarshaler", func(t *testing.T) {
		var got textList

		if err := expandString("go, config", &got); err != nil {
			t.Fatalf("failed to expand: %s", err)
		}

		expected := textList{"go", "config"}
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("got=%+v, expected=%+v", got, expected)
		}
	})

	t.Run("assigns the raw string directly when the target is a string kind", func(t *testing.T) {
		var got name

		if err := expandString("not json", &got); err != nil {
			t.Fatalf("failed to expand: %s", err)
		}

		if got != "not json" {
			t.Fatalf("got=%s, expected=not json", got)
		}
	})

	t.Run("errors when the raw string is not valid JSON for the target", func(t *testing.T) {
		var got database

		if err := expandString("not json", &got); err == nil {
			t.Fatalf("expected an error for a non-JSON string decoded into a struct")
		}
	})
}

func expandFor[T any](t *testing.T, e *Expandable[T], resolvers map[string]Resolver) (bool, error) {
	t.Helper()
	return e.expand(context.Background(), reflect.TypeFor[Expandable[T]](), resolvers)
}

func TestExpandableExpand(t *testing.T) {
	fileResolver := fakeResolver{
		prefix: "file",
		input: map[string]string{
			"db":     `{"host":"prodserver.com","port":4213,"password":"pw"}`,
			"broken": "top-secret-value",
		},
	}
	resolvers := map[string]Resolver{fileResolver.Prefix(): fileResolver}

	t.Run("expands a deferred reference through the matching resolver", func(t *testing.T) {
		e := NewReference[database]("file://db")

		if _, err := expandFor(t, &e, resolvers); err != nil {
			t.Fatalf("failed to expand: %s", err)
		}

		expected := database{Host: "prodserver.com", Port: 4213, Password: "pw"}
		if !reflect.DeepEqual(e.Value, expected) {
			t.Fatalf("got=%+v, expected=%+v", e.Value, expected)
		}
	})

	t.Run("reports nothing to expand when the value was inline", func(t *testing.T) {
		e := NewExpanded(database{Host: "localhost"})

		handled, err := expandFor(t, &e, resolvers)
		if err != nil {
			t.Fatalf("failed to expand: %s", err)
		}

		if handled {
			t.Fatalf("an inline value reported itself as expanded")
		}
	})

	t.Run("returns error when the reference cannot be resolved", func(t *testing.T) {
		e := NewReference[database]("file://missing")

		if _, err := expandFor(t, &e, resolvers); err == nil {
			t.Fatalf("expected an error for an unresolvable reference")
		}
	})

	t.Run("leaves Value untouched when decoding the resolved string fails", func(t *testing.T) {
		e := NewReference[database]("file://broken")

		if _, err := expandFor(t, &e, resolvers); err == nil {
			t.Fatalf("expected an error for a resolved string that is not valid JSON")
		}

		if !reflect.DeepEqual(e.Value, database{}) {
			t.Fatalf("failed expansion left state behind, got=%+v", e.Value)
		}
	})

	t.Run("names the reference and the target type in the decode error", func(t *testing.T) {
		e := NewReference[database]("file://broken")

		_, err := expandFor(t, &e, resolvers)
		if err == nil {
			t.Fatalf("expected an error for a resolved string that is not valid JSON")
		}

		if !strings.Contains(err.Error(), "file://broken") {
			t.Fatalf("error does not name the reference, got=%s", err)
		}
		if !strings.Contains(err.Error(), "comfig.database") {
			t.Fatalf("error does not name the target type, got=%s", err)
		}
	})

	t.Run("keeps the resolved content out of the decode error", func(t *testing.T) {
		e := NewReference[database]("file://broken")

		_, err := expandFor(t, &e, resolvers)
		if err == nil {
			t.Fatalf("expected an error for a resolved string that is not valid JSON")
		}

		if strings.Contains(err.Error(), "top-secret-value") {
			t.Fatalf("error leaks the resolved content, got=%s", err)
		}
	})
}

func TestExpandableFormat(t *testing.T) {
	t.Run("formats as the underlying value once expanded", func(t *testing.T) {
		e := NewExpanded(database{Host: "localhost", Port: 5432, Password: "pw"})

		got := fmt.Sprintf("%+v", e)

		expected := "{Host:localhost Port:5432 Password:pw}"
		if got != expected {
			t.Fatalf("got=%s, expected=%s", got, expected)
		}
	})

	t.Run("formats as the reference while still deferred", func(t *testing.T) {
		e := NewReference[database]("file://db")

		got := fmt.Sprintf("%v", e)

		if got != "file://db" {
			t.Fatalf("got=%s, expected=file://db", got)
		}
	})
}
