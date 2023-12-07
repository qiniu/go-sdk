package main

import (
	"errors"
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
	"gopkg.in/yaml.v3"
)

type (
	JsonType struct {
		String    bool
		Integer   bool
		Float     bool
		Boolean   bool
		Array     *JsonArray
		Struct    *JsonStruct
		Any       bool
		StringMap bool
	}

	JsonArray struct {
		Type          *JsonType `yaml:"type,omitempty"`
		Name          string    `yaml:"name,omitempty"`
		Documentation string    `yaml:"documentation,omitempty"`
	}

	JsonStruct struct {
		Fields        []JsonField `yaml:"fields,omitempty"`
		Name          string      `yaml:"name,omitempty"`
		Documentation string      `yaml:"documentation,omitempty"`
	}

	JsonField struct {
		Type          JsonType           `yaml:"type,omitempty"`
		Key           string             `yaml:"key,omitempty"`
		FieldName     string             `yaml:"field_name,omitempty"`
		Documentation string             `yaml:"documentation,omitempty"`
		Optional      *OptionalType      `yaml:"optional,omitempty"`
		ServiceBucket *ServiceBucketType `yaml:"service_bucket,omitempty"`
	}
)

func (jsonStruct *JsonStruct) addFields(group *jen.Group, includesJsonTag bool) error {
	for _, field := range jsonStruct.Fields {
		code, err := field.Type.AddTypeToStatement(jen.Id(strcase.ToCamel(field.FieldName)))
		if err != nil {
			return err
		}
		if includesJsonTag {
			jsonTag := field.Key
			if field.Optional.ToOptionalType() == OptionalTypeOmitEmpty {
				jsonTag += ",omitempty"
			}
			code = code.Tag(map[string]string{"json": jsonTag})
		}
		if field.Documentation != "" {
			code = code.Comment(field.Documentation)
		}
		group.Add(code)
	}
	return nil
}

func (jsonStruct *JsonStruct) addGetBucketNameFunc(group *jen.Group, structName string) (bool, error) {
	field := jsonStruct.getServiceBucketField()

	if field == nil || field.ServiceBucket.ToServiceBucketType() == ServiceBucketTypeNone {
		return false, nil
	} else if !field.Type.String {
		panic("service bucket field must be string")
	}

	group.Add(jen.Func().
		Params(jen.Id("j").Op("*").Id(structName)).
		Id("getBucketName").
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(jen.String(), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			fieldName := strcase.ToCamel(field.FieldName)
			switch field.ServiceBucket.ToServiceBucketType() {
			case ServiceBucketTypePlainText:
				group.Add(jen.Return(jen.Id("j").Dot(fieldName), jen.Nil()))
			case ServiceBucketTypeEntry:
				group.Add(
					jen.Return(
						jen.Qual("strings", "SplitN").
							Call(jen.Id("j").Dot(fieldName), jen.Lit(":"), jen.Lit(2)).
							Index(jen.Lit(0)),
						jen.Nil(),
					),
				)
			case ServiceBucketTypeUploadToken:
				group.Add(
					jen.If(jen.Id("putPolicy"), jen.Err()).
						Op(":=").
						Qual(PackageNameUpToken, "NewParser").
						Call(jen.Id("j").Dot(fieldName)).
						Dot("GetPutPolicy").
						Call(jen.Id("ctx")).
						Op(";").
						Err().
						Op("!=").
						Nil().
						BlockFunc(func(group *jen.Group) {
							group.Add(jen.Return(jen.Lit(""), jen.Err()))
						}).
						Else().
						BlockFunc(func(group *jen.Group) {
							group.Add(jen.Return(jen.Id("putPolicy").Dot("GetBucketName").Call()))
						}),
				)
			default:
				panic("unknown ServiceBucketType")
			}
		}))
	return true, nil
}

func (jsonType *JsonType) generate(group *jen.Group, options CodeGeneratorOptions) error {
	return jsonType.generateType(group, options, true, func() error {
		if jsonType.Any {
			return jsonType.addAnyJsonMarshalerUnmarshaler(group, options.Name)
		}
		return errors.New("base type could not be top level")
	})
}

func (jsonType *JsonType) generateType(group *jen.Group, options CodeGeneratorOptions, topLevel bool, otherWise func() error) error {
	if s := jsonType.Struct; s != nil {
		if err := s.generate(group, options, topLevel); err != nil {
			return err
		}
		return nil
	} else if a := jsonType.Array; a != nil {
		if err := a.generate(group, options, topLevel); err != nil {
			return err
		}
		return nil
	}
	return otherWise()
}

func (jsonType *JsonType) AddTypeToStatement(statement *jen.Statement) (*jen.Statement, error) {
	if jsonType.String {
		return statement.Add(jen.String()), nil
	} else if jsonType.Integer {
		return statement.Add(jen.Int64()), nil
	} else if jsonType.Float {
		return statement.Add(jen.Float64()), nil
	} else if jsonType.Boolean {
		return statement.Add(jen.Bool()), nil
	} else if jsonType.Any {
		return statement.Add(jen.Interface()), nil
	} else if jsonType.StringMap {
		return statement.Add(jen.Map(jen.String()).String()), nil
	} else if jsonType.Array != nil {
		return statement.Add(jen.Id(strcase.ToCamel(jsonType.Array.Name))), nil
	} else if jsonType.Struct != nil {
		return statement.Add(jen.Id(strcase.ToCamel(jsonType.Struct.Name))), nil
	} else {
		return nil, errors.New("unknown type")
	}
}

func (jsonType *JsonType) ZeroValue() interface{} {
	if jsonType.String {
		return ""
	} else if jsonType.Integer {
		return 0
	} else if jsonType.Float {
		return 0.0
	} else if jsonType.Boolean {
		return false
	} else {
		return nil
	}
}

func (jsonType *JsonType) addAnyJsonMarshalerUnmarshaler(group *jen.Group, structName string) (err error) {
	group.Add(
		jen.Func().
			Params(jen.Id("j").Op("*").Id(structName)).
			Id("MarshalJSON").
			Params().
			Params(jen.Index().Byte(), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.Return(
					jen.Qual("encoding/json", "Marshal").
						Call(jen.Id("j").Dot("Body")),
				))
			}),
	)
	group.Add(
		jen.Func().
			Params(jen.Id("j").Op("*").Id(structName)).
			Id("UnmarshalJSON").
			Params(jen.Id("data").Index().Byte()).
			Params(jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.Var().Id("any").Interface())
				group.Add(
					jen.If(
						jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("any")),
						jen.Err().Op("!=").Nil(),
					).BlockFunc(func(group *jen.Group) {
						group.Return(jen.Err())
					}),
				)
				group.Add(jen.Id("j").Dot("Body").Op("=").Id("any"))
				group.Add(jen.Return(jen.Nil()))
			}),
	)
	return
}

func (jsonArray *JsonArray) addFields(group *jen.Group) error {
	code := jen.Id(strcase.ToCamel(jsonArray.Name)).Id(jsonArray.Name)
	if jsonArray.Documentation != "" {
		code = code.Comment(jsonArray.Documentation)
	}
	group.Add(code)
	return nil
}

func (jsonArray *JsonArray) generate(group *jen.Group, options CodeGeneratorOptions, topLevel bool) (err error) {
	if err = jsonArray.Type.generateType(group, CodeGeneratorOptions{}, false, func() error {
		return nil
	}); err != nil {
		return
	}

	if jsonArray.Documentation != "" {
		group.Add(jen.Comment(jsonArray.Documentation))
	}
	code := jen.Type().Id(strcase.ToCamel(jsonArray.Name))
	if !topLevel {
		code = code.Op("=")
	}
	code = code.Index()
	code, err = jsonArray.Type.AddTypeToStatement(code)
	if err != nil {
		return
	}
	group.Add(code)
	if topLevel {
		if err = jsonArray.addJsonMarshalerUnmarshaler(group, options.Name); err != nil {
			return
		}
	}

	return
}

func (jsonArray *JsonArray) addJsonMarshalerUnmarshaler(group *jen.Group, structName string) (err error) {
	fieldName := strcase.ToCamel(jsonArray.Name)
	group.Add(
		jen.Func().
			Params(jen.Id("j").Op("*").Id(structName)).
			Id("MarshalJSON").
			Params().
			Params(jen.Index().Byte(), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.Return(
					jen.Qual("encoding/json", "Marshal").
						Call(jen.Id("j").Dot(fieldName)),
				))
			}),
	)
	group.Add(
		jen.Func().
			Params(jen.Id("j").Op("*").Id(structName)).
			Id("UnmarshalJSON").
			Params(jen.Id("data").Index().Byte()).
			Params(jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.Var().Id("array").Id(fieldName))
				group.Add(
					jen.If(
						jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("array")),
						jen.Err().Op("!=").Nil(),
					).BlockFunc(func(group *jen.Group) {
						group.Return(jen.Err())
					}),
				)
				group.Add(jen.Id("j").Dot(fieldName).Op("=").Id("array"))
				group.Add(jen.Return(jen.Nil()))
			}),
	)
	return
}

func (jsonStruct *JsonStruct) generate(group *jen.Group, options CodeGeneratorOptions, topLevel bool) (err error) {
	for _, field := range jsonStruct.Fields {
		if err = field.Type.generateType(group, CodeGeneratorOptions{Name: field.FieldName, Documentation: field.Documentation}, false, func() error {
			return nil
		}); err != nil {
			return
		}
	}

	opts := make([]CodeGeneratorOptions, 0, 2)
	if options.Name != "" {
		opts = append(opts, CodeGeneratorOptions{Name: strcase.ToCamel(options.Name), Documentation: options.Documentation})
	}
	if jsonStruct.Name != "" && strcase.ToCamel(options.Name) != strcase.ToCamel(jsonStruct.Name) {
		opts = append(opts, CodeGeneratorOptions{Name: strcase.ToCamel(jsonStruct.Name), Documentation: jsonStruct.Documentation})
	}
	if len(opts) == 0 {
		return errors.New("unknown struct name")
	}

	if !topLevel {
		if opts[0].Documentation != "" {
			group.Add(jen.Comment(opts[0].Documentation))
		}
		group.Add(jen.Type().Id(opts[0].Name).StructFunc(func(group *jen.Group) {
			err = jsonStruct.addFields(group, false)
		}))
		if err != nil {
			return
		}
	}

	if len(opts) > 1 {
		if opts[1].Documentation != "" {
			group.Add(jen.Comment(opts[1].Documentation))
		}
		group.Add(jen.Type().Id(strcase.ToCamel(opts[1].Name)).Op("=").Id(strcase.ToCamel(opts[0].Name)))
	}

	if err = jsonStruct.addJsonMarshalerUnmarshaler(group, opts[0].Name); err != nil {
		return
	}
	if err = jsonStruct.generateValidateFunc(group, opts[0].Name); err != nil {
		return
	}

	return
}

func (jsonStruct *JsonStruct) addJsonMarshalerUnmarshaler(group *jen.Group, structName string) (err error) {
	group.Add(jen.Type().Id("json" + structName).StructFunc(func(group *jen.Group) {
		err = jsonStruct.addFields(group, true)
	}))
	if err != nil {
		return
	}
	group.Add(
		jen.Func().
			Params(jen.Id("j").Op("*").Id(structName)).
			Id("MarshalJSON").
			Params().
			Params(jen.Index().Byte(), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.If(
					jen.Err().Op(":=").Id("j").Dot("validate").Call(),
					jen.Err().Op("!=").Nil(),
				).BlockFunc(func(group *jen.Group) {
					group.Return(jen.Nil(), jen.Err())
				}))
				group.Add(jen.Return(
					jen.Qual("encoding/json", "Marshal").
						Call(
							jen.Op("&").Id("json" + structName).
								ValuesFunc(func(group *jen.Group) {
									for _, field := range jsonStruct.Fields {
										fieldName := strcase.ToCamel(field.FieldName)
										group.Add(jen.Id(fieldName).Op(":").Id("j").Dot(fieldName))
									}
								}),
						),
				))
			}),
	)
	group.Add(
		jen.Func().
			Params(jen.Id("j").Op("*").Id(structName)).
			Id("UnmarshalJSON").
			Params(jen.Id("data").Index().Byte()).
			Params(jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.Var().Id("nj").Id("json" + structName))
				group.Add(
					jen.If(
						jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("nj")),
						jen.Err().Op("!=").Nil(),
					).BlockFunc(func(group *jen.Group) {
						group.Return(jen.Err())
					}),
				)
				for _, field := range jsonStruct.Fields {
					fieldName := strcase.ToCamel(field.FieldName)
					group.Add(jen.Id("j").Dot(fieldName).Op("=").Id("nj").Dot(fieldName))
				}
				group.Add(jen.Return(jen.Nil()))
			}),
	)
	return
}

func (jsonStruct *JsonStruct) generateValidateFunc(group *jen.Group, structName string) error {
	group.Add(jen.Func().
		Params(jen.Id("j").Op("*").Id(structName)).
		Id("validate").
		Params().
		Params(jen.Error()).
		BlockFunc(func(group *jen.Group) {
			for _, field := range jsonStruct.Fields {
				if field.Optional.ToOptionalType() == OptionalTypeRequired {
					var cond *jen.Statement
					fieldName := strcase.ToCamel(field.FieldName)
					if field.Type.String || field.Type.Integer || field.Type.Float {
						cond = jen.Id("j").Dot(fieldName).Op("==").Lit(field.Type.ZeroValue())
					} else if field.Type.Boolean {
						// do nothing
					} else if field.Type.Array != nil {
						cond = jen.Len(jen.Id("j").Dot(fieldName)).Op("==").Lit(0)
					} else if field.Type.Struct != nil {
						// do nothing
					} else {
						cond = jen.Id("j").Dot(fieldName).Op("==").Nil()
					}
					if cond != nil {
						group.Add(jen.If(cond).BlockFunc(func(group *jen.Group) {
							group.Add(jen.Return(
								jen.Qual(PackageNameErrors, "MissingRequiredFieldError").
									ValuesFunc(func(group *jen.Group) {
										group.Add(jen.Id("Name").Op(":").Lit(fieldName))
									}),
							))
						}))
					}
					if arrayField := field.Type.Array; arrayField != nil {
						if arrayField.Type.Struct != nil {
							group.Add(
								jen.For(jen.List(jen.Id("_"), jen.Id("value")).Op(":=").Range().Id("j").Dot(fieldName)).
									BlockFunc(func(group *jen.Group) {
										group.Add(
											jen.If(
												jen.Err().Op(":=").Id("value").Dot("validate").Call(),
												jen.Err().Op("!=").Nil(),
											).BlockFunc(func(group *jen.Group) {
												group.Return(jen.Err())
											}),
										)
									}),
							)
						}
					} else if field.Type.Struct != nil {
						group.Add(
							jen.If(
								jen.Err().Op(":=").Id("j").Dot(fieldName).Dot("validate").Call(),
								jen.Err().Op("!=").Nil(),
							).BlockFunc(func(group *jen.Group) {
								group.Return(jen.Err())
							}),
						)
					}
				}
			}
			group.Add(jen.Return(jen.Nil()))
		}))
	return nil
}

func (jsonStruct *JsonStruct) getServiceBucketField() *JsonField {
	var serviceBucketField *JsonField = nil

	for i := range jsonStruct.Fields {
		if jsonStruct.Fields[i].ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
			if serviceBucketField == nil {
				serviceBucketField = &jsonStruct.Fields[i]
			} else {
				panic("multiple service bucket fields")
			}
		}
	}
	return serviceBucketField
}

func (jsonType *JsonType) UnmarshalYAML(value *yaml.Node) error {
	switch value.ShortTag() {
	case "!!str":
		switch value.Value {
		case "string":
			jsonType.String = true
		case "integer":
			jsonType.Integer = true
		case "float":
			jsonType.Float = true
		case "boolean":
			jsonType.Boolean = true
		case "any":
			jsonType.Any = true
		case "string_map":
			jsonType.StringMap = true
		default:
			return fmt.Errorf("unknown json type: %s", value.Value)
		}
		return nil
	case "!!map":
		switch value.Content[0].Value {
		case "array":
			return value.Content[1].Decode(&jsonType.Array)
		case "struct":
			return value.Content[1].Decode(&jsonType.Struct)
		default:
			return fmt.Errorf("unknown json type: %s", value.Content[0].Value)
		}
	default:
		return fmt.Errorf("unknown json type: %s", value.ShortTag())
	}
}
