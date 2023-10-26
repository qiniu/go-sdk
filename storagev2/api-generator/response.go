package main

import (
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
	"gopkg.in/yaml.v3"
)

type (
	ApiResponseDescription struct {
		Body *ResponseBody `yaml:"body,omitempty"`
	}

	ResponseBody struct {
		Json             *JsonType
		BinaryDataStream bool
	}
)

func (response *ApiResponseDescription) Generate(group *jen.Group, opts CodeGeneratorOptions) error {
	if opts.Documentation != "" {
		group.Add(jen.Comment(opts.Documentation))
	}
	group.Add(jen.Type().Id(strcase.ToCamel(opts.Name)).StructFunc(func(group *jen.Group) {
		if body := response.Body; body != nil {
			if body.Json != nil {
				group.Add(jen.Id("Body").Id("ResponseBody"))
			} else if body.BinaryDataStream {
				group.Add(jen.Id("Body").Qual("io", "ReadCloser"))
			}
		}
	}))
	return nil
}

func (body *ResponseBody) UnmarshalYAML(value *yaml.Node) error {
	switch value.ShortTag() {
	case "!!str":
		switch value.Value {
		case "binary_data_stream":
			body.BinaryDataStream = true
			return nil
		default:
			return fmt.Errorf("unknown response body type: %s", value.Value)
		}
	case "!!map":
		switch value.Content[0].Value {
		case "json":
			return value.Content[1].Decode(&body.Json)
		default:
			return fmt.Errorf("unknown response body type: %s", value.Content[0].Value)
		}
	default:
		return fmt.Errorf("unknown response body type: %s", value.ShortTag())
	}
}
