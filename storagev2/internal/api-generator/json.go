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

func (jsonType *JsonType) Generate(group *jen.Group, options CodeGeneratorOptions) error {
	return jsonType.generate(group, options, func() error {
		if jsonType.Any {
			if options.Documentation != "" {
				group.Add(jen.Comment(options.Documentation))
			}
			group.Add(jen.Type().Id(strcase.ToCamel(options.Name)).Op("=").Interface())
			return nil
		}
		return errors.New("base type could not be top level")
	})
}

func (jsonType *JsonType) GenerateAliasesFor(group *jen.Group, structName, fieldName string) error {
	if s := jsonType.Struct; s != nil {
		return s.GenerateAliasesFor(group, structName, fieldName)
	}
	return nil
}

func (jsonType *JsonType) generate(group *jen.Group, options CodeGeneratorOptions, otherWise func() error) error {
	if s := jsonType.Struct; s != nil {
		if err := s.Generate(group, options); err != nil {
			return err
		}
		return nil
	} else if a := jsonType.Array; a != nil {
		if err := a.Generate(group, options); err != nil {
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

func (jsonArray *JsonArray) Generate(group *jen.Group, options CodeGeneratorOptions) error {
	if err := jsonArray.Type.generate(group, CodeGeneratorOptions{}, func() error {
		return nil
	}); err != nil {
		return err
	}

	if jsonArray.Documentation != "" {
		group.Add(jen.Comment(jsonArray.Documentation))
	}
	code := jen.Type().Id(strcase.ToCamel(jsonArray.Name)).Op("=").Index()
	code, err := jsonArray.Type.AddTypeToStatement(code)
	if err != nil {
		return err
	}
	group.Add(code)

	if options.Name != "" && strcase.ToCamel(options.Name) != strcase.ToCamel(jsonArray.Name) {
		if options.Documentation != "" {
			group.Add(jen.Comment(options.Documentation))
		}
		group.Add(jen.Type().Id(strcase.ToCamel(options.Name)).Op("=").Id(strcase.ToCamel(jsonArray.Name)))
	}

	return nil
}

func (jsonStruct *JsonStruct) Generate(group *jen.Group, options CodeGeneratorOptions) error {
	for _, field := range jsonStruct.Fields {
		if err := field.Type.generate(group, CodeGeneratorOptions{Name: field.FieldName, Documentation: field.Documentation}, func() error {
			return nil
		}); err != nil {
			return err
		}
	}

	var err error
	opts := make([]CodeGeneratorOptions, 0, 2)

	if jsonStruct.Name != "" {
		opts = append(opts, CodeGeneratorOptions{Name: strcase.ToCamel(jsonStruct.Name), Documentation: jsonStruct.Documentation})
	}
	if options.Name != "" && strcase.ToCamel(options.Name) != strcase.ToCamel(jsonStruct.Name) {
		opts = append(opts, CodeGeneratorOptions{Name: strcase.ToCamel(options.Name), Documentation: options.Documentation})
	}

	if len(opts) == 0 {
		return errors.New("unknown struct name")
	}
	group.Add(
		jen.Type().Id("inner" + opts[0].Name).StructFunc(func(group *jen.Group) {
			for _, field := range jsonStruct.Fields {
				jsonTag := field.Key
				if field.Optional.ToOptionalType() == OptionalTypeOmitEmpty {
					jsonTag += ",omitempty"
				}
				tag := map[string]string{"json": jsonTag}
				code, e := field.Type.AddTypeToStatement(jen.Id(strcase.ToCamel(field.FieldName)))
				if e != nil {
					err = e
					return
				}
				code = code.Tag(tag)
				if field.Documentation != "" {
					code = code.Comment(field.Documentation)
				}
				group.Add(code)
			}
		}),
	)

	if opts[0].Documentation != "" {
		group.Add(jen.Comment(opts[0].Documentation))
	}
	group.Add(
		jen.Type().Id(opts[0].Name).StructFunc(func(group *jen.Group) {
			group.Add(jen.Id("inner").Id("inner" + opts[0].Name))
		}),
	)
	if err != nil {
		return err
	}

	for _, field := range jsonStruct.Fields {
		if err = jsonStruct.generateGetterFunc(group, field, opts[0]); err != nil {
			return err
		}
		if err = jsonStruct.generateSetterFunc(group, field, opts[0]); err != nil {
			return err
		}
	}
	if code := jsonStruct.generateServiceBucketField(opts[0]); code != nil {
		group.Add(code)
	}
	group.Add(jsonStruct.generateMarlshalerFunc(opts[0]))
	group.Add(jsonStruct.generateUnmarlshalerFunc(opts[0]))

	group.Add(jen.Comment("//lint:ignore U1000 may not call it"))
	group.Add(jsonStruct.generateValidateFunc(opts[0]))

	if len(opts) > 1 {
		if opts[1].Documentation != "" {
			group.Add(jen.Comment(opts[1].Documentation))
		}
		group.Add(jen.Type().Id(strcase.ToCamel(opts[1].Name)).Op("=").Id(strcase.ToCamel(opts[0].Name)))
	}
	return nil
}

func (jsonStruct *JsonStruct) GenerateAliasesFor(group *jen.Group, structName, fieldName string) error {
	for _, field := range jsonStruct.Fields {
		if err := jsonStruct.generateAliasGetterFunc(group, field, structName, fieldName); err != nil {
			return err
		}
		if err := jsonStruct.generateAliasSetterFunc(group, field, structName, fieldName); err != nil {
			return err
		}
	}
	return nil
}

func (jsonStruct *JsonStruct) generateMarlshalerFunc(options CodeGeneratorOptions) jen.Code {
	return jen.Func().
		Params(jen.Id("j").Op("*").Id(options.Name)).
		Id("MarshalJSON").
		Params().
		Params(jen.Index().Byte(), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Return(jen.Qual("encoding/json", "Marshal").Call(jen.Op("&").Id("j").Dot("inner"))))
		})
}

func (jsonStruct *JsonStruct) generateUnmarlshalerFunc(options CodeGeneratorOptions) jen.Code {
	return jen.Func().
		Params(jen.Id("j").Op("*").Id(options.Name)).
		Id("UnmarshalJSON").
		Params(jen.Id("data").Index().Byte()).
		Params(jen.Error()).
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Return(jen.Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("j").Dot("inner"))))
		})
}

func (jsonStruct *JsonStruct) generateValidateFunc(options CodeGeneratorOptions) jen.Code {
	return jen.Func().
		Params(jen.Id("j").Op("*").Id(options.Name)).
		Id("validate").
		Params().
		Params(jen.Error()).
		BlockFunc(func(group *jen.Group) {
			for _, field := range jsonStruct.Fields {
				if field.Optional.ToOptionalType() == OptionalTypeRequired {
					var cond *jen.Statement
					fieldName := strcase.ToCamel(field.FieldName)
					if field.Type.String || field.Type.Integer || field.Type.Float {
						cond = jen.Id("j").Dot("inner").Dot(fieldName).Op("==").Lit(field.Type.ZeroValue())
					} else if field.Type.Boolean {
						// do nothing
					} else if field.Type.Array != nil {
						cond = jen.Len(jen.Id("j").Dot("inner").Dot(fieldName)).Op(">").Lit(0)
					} else if field.Type.Struct != nil {
						// do nothing
					} else {
						cond = jen.Id("j").Dot("inner").Dot(fieldName).Op("!=").Nil()
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
								jen.For(jen.List(jen.Id("_"), jen.Id("value")).Op(":=").Range().Id("j").Dot("inner").Dot(fieldName)).
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
								jen.Err().Op(":=").Id("j").Dot("inner").Dot(fieldName).Dot("validate").Call(),
								jen.Err().Op("!=").Nil(),
							).BlockFunc(func(group *jen.Group) {
								group.Return(jen.Err())
							}),
						)
					}
				}
			}
			group.Add(jen.Return(jen.Nil()))
		})
}

func (jsonStruct *JsonStruct) generateAliasGetterFunc(group *jen.Group, field JsonField, structName, fieldName string) error {
	if field.Documentation != "" {
		group.Add(jen.Comment(field.Documentation))
	}
	code := jen.Func().
		Params(jen.Id("request").Op("*").Id(structName)).
		Id(makeGetterMethodName(field.FieldName)).
		Params()
	code, err := field.Type.AddTypeToStatement(code)
	if err != nil {
		return err
	}
	code = code.BlockFunc(func(group *jen.Group) {
		group.Add(jen.Return(jen.Id("request").Dot(fieldName).Dot(makeGetterMethodName(field.FieldName)).Call()))
	})
	group.Add(code)
	return nil
}

func (jsonStruct *JsonStruct) generateAliasSetterFunc(group *jen.Group, field JsonField, structName, fieldName string) error {
	if field.Documentation != "" {
		group.Add(jen.Comment(field.Documentation))
	}
	params, err := field.Type.AddTypeToStatement(jen.Id("value"))
	if err != nil {
		return err
	}
	group.Add(jen.Func().
		Params(jen.Id("request").Op("*").Id(structName)).
		Id(makeSetterMethodName(field.FieldName)).
		Params(params).
		Params(jen.Op("*").Id(structName)).
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Id("request").Dot(fieldName).Dot(makeSetterMethodName(field.FieldName)).Call(jen.Id("value")))
			group.Add(jen.Return(jen.Id("request")))
		}))
	return nil
}

func (jsonStruct *JsonStruct) generateGetterFunc(group *jen.Group, field JsonField, options CodeGeneratorOptions) error {
	if field.Documentation != "" {
		group.Add(jen.Comment(field.Documentation))
	}
	code := jen.Func().
		Params(jen.Id("j").Op("*").Id(options.Name)).
		Id(makeGetterMethodName(field.FieldName)).
		Params()
	code, err := field.Type.AddTypeToStatement(code)
	if err != nil {
		return err
	}
	code = code.BlockFunc(func(group *jen.Group) {
		group.Add(jen.Return(jen.Id("j").Dot("inner").Dot(strcase.ToCamel(field.FieldName))))
	})
	group.Add(code)
	return nil
}

func (jsonStruct *JsonStruct) generateSetterFunc(group *jen.Group, field JsonField, options CodeGeneratorOptions) error {
	if field.Documentation != "" {
		group.Add(jen.Comment(field.Documentation))
	}
	params, err := field.Type.AddTypeToStatement(jen.Id("value"))
	if err != nil {
		return err
	}
	group.Add(jen.Func().
		Params(jen.Id("j").Op("*").Id(options.Name)).
		Id(makeSetterMethodName(field.FieldName)).
		Params(params).
		Params(jen.Op("*").Id(options.Name)).
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Id("j").Dot("inner").Dot(strcase.ToCamel(field.FieldName)).Op("=").Id("value"))
			group.Add(jen.Return(jen.Id("j")))
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

func (jsonStruct *JsonStruct) generateServiceBucketField(options CodeGeneratorOptions) jen.Code {
	field := jsonStruct.getServiceBucketField()
	if field == nil || field.ServiceBucket.ToServiceBucketType() == ServiceBucketTypeNone {
		return nil
	} else if !field.Type.String {
		panic("service bucket field must be string")
	}
	return jen.Func().
		Params(jen.Id("j").Op("*").Id(options.Name)).
		Id("getBucketName").
		Params().
		Params(jen.String(), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			switch field.ServiceBucket.ToServiceBucketType() {
			case ServiceBucketTypePlainText:
				group.Add(jen.Return(jen.Id("j").Dot("inner").Dot(strcase.ToCamel(field.FieldName)), jen.Nil()))
			case ServiceBucketTypeEntry:
				group.Add(
					jen.Return(
						jen.Qual("strings", "SplitN").
							Call(jen.Id("j").Dot("inner").Dot(strcase.ToCamel(field.FieldName)), jen.Lit(":"), jen.Lit(2)).
							Index(jen.Lit(0)),
						jen.Nil(),
					),
				)
			case ServiceBucketTypeUploadToken:
				group.Add(
					jen.If(jen.Id("putPolicy"), jen.Err()).
						Op(":=").
						Qual(PackageNameUpToken, "NewParser").
						Call(jen.Id("j").Dot("inner").Dot(strcase.ToCamel(field.FieldName))).
						Dot("RetrievePutPolicy").
						Call(jen.Qual("context", "Background").Call()).
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
		})
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
