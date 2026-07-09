package comfig

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

func resolve(ctx context.Context, config any, resolvers map[string]Resolver) error {
	root := reflect.ValueOf(config)
	if root.Kind() != reflect.Pointer || root.IsNil() {
		return fmt.Errorf("config must be a non-nil pointer to a struct, got %T", config)
	}

	root = root.Elem()
	if root.Kind() != reflect.Struct {
		return fmt.Errorf("config must be a struct, got %s", root.Kind())
	}

	var walk func(v reflect.Value) error
	walk = func(v reflect.Value) error {
		switch v.Kind() {
		case reflect.Pointer:
			if v.IsNil() {
				return nil
			}
			return walk(v.Elem())

		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				field := v.Field(i)
				if !field.CanSet() {
					continue
				}
				if err := walk(field); err != nil {
					return fmt.Errorf("%s: %w", v.Type().Field(i).Name, err)
				}
			}
			return nil

		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				if err := walk(v.Index(i)); err != nil {
					return fmt.Errorf("[%d]: %w", i, err)
				}
			}
			return nil

		case reflect.String:
			if !v.CanSet() {
				return nil
			}
			resolved, err := resolveString(ctx, v.String(), resolvers)
			if err != nil {
				return err
			}
			v.SetString(resolved)
			return nil

		case reflect.Interface:
			if v.IsNil() || !v.CanSet() {
				return nil
			}
			cp := reflect.New(v.Elem().Type()).Elem()
			cp.Set(v.Elem())
			if err := walk(cp); err != nil {
				return err
			}
			v.Set(cp)
			return nil

		case reflect.Map:
			if v.IsNil() {
				return nil
			}
			for _, key := range v.MapKeys() {
				val := reflect.New(v.Type().Elem()).Elem()
				val.Set(v.MapIndex(key))
				if err := walk(val); err != nil {
					return fmt.Errorf("[%v]: %w", key, err)
				}
				v.SetMapIndex(key, val)
			}
			return nil

		default:
			return nil
		}
	}

	return walk(root)
}

func resolveString(ctx context.Context, value string, resolvers map[string]Resolver) (string, error) {
	prefix, rest := splitPrefix(value)
	if prefix == "" {
		return value, nil
	}

	resolver, exists := resolvers[prefix]
	if !exists {
		return value, nil
	}

	resolved, err := resolver.Resolve(ctx, rest)
	if err != nil {
		return "", fmt.Errorf("could not resolve %s: %w", value, err)
	}

	return resolved, nil
}

func splitPrefix(value string) (prefix, rest string) {
	idx := strings.Index(value, "://")
	if idx <= 0 {
		return "", value
	}
	return value[:idx], value[idx+len("://"):]
}
