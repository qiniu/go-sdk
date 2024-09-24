package main

import (
	"errors"
	"fmt"

	"github.com/dave/jennifer/jen"
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

func (response *ApiResponseDescription) generate(group *jen.Group, opts CodeGeneratorOptions) (err error) {
	if opts.Documentation != "" {
		group.Add(jen.Comment(opts.Documentation))
	}
	group.Add(jen.Type().
		Id(opts.camelCaseName()).
		StructFunc(func(group *jen.Group) {
			err = response.addFields(group)
		}))
	if err != nil {
		return
	}

	if body := response.Body; body != nil {
		if bodyJson := body.Json; bodyJson != nil {
			err = bodyJson.generate(group, opts)
		}
	}

	return
}

func (response *ApiResponseDescription) addFields(group *jen.Group) (err error) {
	if body := response.Body; body != nil {
		if bodyJson := body.Json; bodyJson != nil {
			if jsonStruct := bodyJson.Struct; jsonStruct != nil {
				if err = jsonStruct.addFields(group, false); err != nil {
					return
				}
			} else if jsonArray := bodyJson.Array; jsonArray != nil {
				if err = jsonArray.addFields(group); err != nil {
					return
				}
			} else if bodyJson.Any {
				group.Add(jen.Id("Body").Interface())
			} else if bodyJson.StringMap {
				group.Add(jen.Id("Body").Map(jen.String()).String())
			} else {
				return errors.New("response body should be struct or array")
			}
		} else if body.BinaryDataStream {
			group.Add(jen.Id("Body").Qual("io", "ReadCloser"))
		}
	}
	return
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
