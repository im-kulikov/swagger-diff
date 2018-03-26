package swagger

import (
	"encoding/json"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
)

// LoadFile yaml to swagger
func LoadFile(file string) (*spec.Swagger, error) {
	doc, err := swag.YAMLDoc(file)
	if err != nil {
		return nil, err
	}

	var result = new(spec.Swagger)

	if err = json.Unmarshal(doc, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// LoadContent yaml to swagger
func LoadContent(content string) (*spec.Swagger, error) {
	doc, err := swag.BytesToYAMLDoc([]byte(content))
	if err != nil {
		return nil, err
	}

	jsonData, err := swag.YAMLToJSON(doc)
	if err != nil {
		return nil, err
	}

	var result = new(spec.Swagger)

	if err = json.Unmarshal(jsonData, &result); err != nil {
		return nil, err
	}

	return result, nil
}
