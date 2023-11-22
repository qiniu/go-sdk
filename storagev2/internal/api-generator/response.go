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
				group.Add(jen.Id("body").Id("ResponseBody"))
			} else if body.BinaryDataStream {
				group.Add(jen.Id("body").Qual("io", "ReadCloser"))
			}
		}
	}))
	response.generateGetters(group, opts)
	response.generateSetters(group, opts)
	return nil
}

func (response *ApiResponseDescription) generateGetters(group *jen.Group, opts CodeGeneratorOptions) {
	structName := strcase.ToCamel(opts.Name)
	if body := response.Body; body != nil {
		var (
			returnType    jen.Code
			returnPointer bool
		)
		if json := body.Json; json != nil {
			if json.Struct != nil {
				returnType = jen.Op("*").Id("ResponseBody")
				returnPointer = true
			} else {
				returnType = jen.Id("ResponseBody")
			}
		} else if body.BinaryDataStream {
			returnType = jen.Qual("io", "ReadCloser")
		}
		group.Add(jen.Comment("获取请求体"))
		group.Add(
			jen.Func().
				Params(jen.Id("response").Op("*").Id(structName)).
				Id("GetBody").
				Params().
				Params(returnType).
				BlockFunc(func(group *jen.Group) {
					if returnPointer {
						group.Return(jen.Op("&").Id("response").Dot("body"))
					} else {
						group.Return(jen.Id("response").Dot("body"))
					}
				}),
		)
	}
}

func (response *ApiResponseDescription) generateSetters(group *jen.Group, opts CodeGeneratorOptions) {
	structName := strcase.ToCamel(opts.Name)
	if body := response.Body; body != nil {
		var returnType jen.Code
		if body.Json != nil {
			returnType = jen.Id("ResponseBody")
		} else if body.BinaryDataStream {
			returnType = jen.Qual("io", "ReadCloser")
		}
		group.Add(jen.Comment("设置请求体"))
		group.Add(
			jen.Func().
				Params(jen.Id("response").Op("*").Id(structName)).
				Id("SetBody").
				Params(jen.Id("body").Add(returnType)).
				Params(jen.Op("*").Id(structName)).
				BlockFunc(func(group *jen.Group) {
					group.Add(jen.Id("response").Dot("body").Op("=").Id("body"))
					group.Add(jen.Return(jen.Id("response")))
				}),
		)
	}
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
