package utils

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/santhosh-tekuri/jsonschema"
)

var (
	ValidateSchemaError = errors.New("Error validating schema")
)

func CompileJsonSchema(schemaJSON *json.RawMessage) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()

	if err := compiler.AddResource(
		"schema.json",
		bytes.NewReader(*schemaJSON),
	); err != nil {
		return nil, err
	}

	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func ValidateRawMessageWithSchema(
	schemaJSON json.RawMessage,
	payload json.RawMessage,
) error {
	schema, err := CompileJsonSchema(&schemaJSON)
	if err != nil {
		return err
	}

	return schema.Validate(bytes.NewReader(payload))
}
