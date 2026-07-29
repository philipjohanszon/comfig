package comfig

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

type Expander interface {
	Expand(raw string) error
}

type Expandable[T any] struct {
	Value      T
	raw        *string
	isExpanded bool
}

func NewExpanded[T any](value T) Expandable[T] {
	return Expandable[T]{Value: value, raw: nil, isExpanded: true}
}

func NewReference[T any](raw string) Expandable[T] {
	return Expandable[T]{raw: &raw, isExpanded: false}
}

func (e *Expandable[T]) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if string(trimmed) == "null" {
		return nil
	}

	*e = Expandable[T]{}

	if len(trimmed) > 0 && trimmed[0] == '"' {
		var raw string
		if err := json.Unmarshal(trimmed, &raw); err != nil {
			return err
		}

		e.Value = *new(T)
		e.raw = &raw
		e.isExpanded = false
		return nil
	}

	var value T
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return fmt.Errorf("unmarshal into %s: %w", reflect.TypeFor[T](), err)
	}

	e.Value = value
	e.raw = nil
	e.isExpanded = true
	return nil
}

func (e Expandable[T]) Format(f fmt.State, verb rune) {
	if e.isExpanded {
		fmt.Fprintf(f, fmt.FormatString(f, verb), e.Value)
		return
	}

	if e.raw == nil {
		fmt.Fprintf(f, "nil")
		return
	}

	fmt.Fprintf(f, fmt.FormatString(f, verb), *e.raw)
}

func expandValue(ctx context.Context, v reflect.Value, resolvers map[string]Resolver) (wasExpanded bool, err error) {
	node, ok := v.Addr().Interface().(interface {
		expand(ctx context.Context, self reflect.Type, resolvers map[string]Resolver) (bool, error)
	})
	if !ok {
		return false, nil
	}

	return node.expand(ctx, v.Type(), resolvers)
}

func (e *Expandable[T]) expand(ctx context.Context, self reflect.Type, resolvers map[string]Resolver) (wasExpanded bool, err error) {
	if e.raw == nil && !e.isExpanded {
		return false, errors.New("cannot expand when there is no unresolved value and no inline value")
	}

	if e.raw != nil && e.isExpanded {
		return true, nil
	}

	if self != reflect.TypeFor[Expandable[T]]() || e.isExpanded {
		return false, nil
	}

	resolved, err := resolveString(ctx, *e.raw, resolvers)
	if err != nil {
		return false, err
	}

	var value T
	if err = expandString(resolved, &value); err != nil {
		return false, fmt.Errorf("expand %q into %s: %w", *e.raw, reflect.TypeFor[T](), err)
	}

	*e = Expandable[T]{Value: value, raw: e.raw, isExpanded: true}
	return true, nil
}

func expandString[T any](raw string, target *T) error {
	switch decoder := any(target).(type) {
	case Expander:
		return decoder.Expand(raw)
	case encoding.TextUnmarshaler:
		return decoder.UnmarshalText([]byte(raw))
	}

	if value := reflect.ValueOf(target).Elem(); value.Kind() == reflect.String {
		value.SetString(raw)
		return nil
	}

	return json.Unmarshal([]byte(raw), target)
}
