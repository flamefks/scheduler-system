package config

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const redactedValue = "***REDACTED***"

var (
	postgresURLPattern = regexp.MustCompile(`postgres(?:ql)?://[^\s"']+`)
	passwordPattern    = regexp.MustCompile(`(?i)(password|passwd|token|secret)=([^ \t\n\r;]+)`)
)

func RedactConfigYAML(v any) string {
	redacted := redactValue(reflect.ValueOf(v))
	out, err := yaml.Marshal(redacted)
	if err != nil {
		return fmt.Sprintf("<redact config: %v>", err)
	}
	return string(out)
}

func SafeDSN(raw string) string {
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil {
		return redactedValue
	}

	if u.User != nil {
		username := u.User.Username()
		if _, hasPassword := u.User.Password(); hasPassword {
			u.User = url.UserPassword(username, redactedValue)
		}
	}

	return u.String()
}

func RedactText(text string) string {
	text = postgresURLPattern.ReplaceAllStringFunc(text, SafeDSN)
	return passwordPattern.ReplaceAllString(text, `${1}=`+redactedValue)
}

func redactValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}

	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		return redactValue(v.Elem())
	}

	switch v.Kind() {
	case reflect.Struct:
		result := map[string]any{}
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue
			}

			key := yamlFieldName(field)
			if key == "-" {
				continue
			}

			if isSecretField(key, field.Name) {
				result[key] = redactScalar(v.Field(i))
				continue
			}

			result[key] = redactValue(v.Field(i))
		}
		return result
	case reflect.Map:
		result := map[string]any{}
		iter := v.MapRange()
		for iter.Next() {
			key := fmt.Sprint(iter.Key().Interface())
			if isSecretField(key, key) {
				result[key] = redactScalar(iter.Value())
				continue
			}
			result[key] = redactValue(iter.Value())
		}
		return result
	case reflect.Slice, reflect.Array:
		items := make([]any, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			items = append(items, redactValue(v.Index(i)))
		}
		return items
	default:
		return v.Interface()
	}
}

func yamlFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("yaml")
	if tag == "" {
		return field.Name
	}
	return strings.Split(tag, ",")[0]
}

func isSecretField(key string, fieldName string) bool {
	normalized := strings.ToLower(key + "_" + fieldName)
	return strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "passwd") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "dsn") ||
		key == "url"
}

func redactScalar(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		return redactScalar(v.Elem())
	}
	if v.Kind() != reflect.String {
		return redactedValue
	}
	return SafeDSN(v.String())
}
