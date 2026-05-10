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

func ValidateRawMessageWithSchema(
	schemaJSON json.RawMessage,
	payload json.RawMessage,
) error {
	compiler := jsonschema.NewCompiler()

	if err := compiler.AddResource(
		"schema.json",
		bytes.NewReader(schemaJSON),
	); err != nil {
		return err
	}

	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return err
	}

	return schema.Validate(bytes.NewReader(payload))
}
