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
		CamelCaseName string    `yaml:"camel_case_name,omitempty"`
		SnakeCaseName string    `yaml:"snake_case_name,omitempty"`
		Documentation string    `yaml:"documentation,omitempty"`
	}

	JsonStruct struct {
		Fields        []JsonField `yaml:"fields,omitempty"`
		Name          string      `yaml:"name,omitempty"`
		CamelCaseName string      `yaml:"camel_case_name,omitempty"`
		SnakeCaseName string      `yaml:"snake_case_name,omitempty"`
		Documentation string      `yaml:"documentation,omitempty"`
	}

	JsonField struct {
		Type               JsonType           `yaml:"type,omitempty"`
		Key                string             `yaml:"key,omitempty"`
		FieldName          string             `yaml:"field_name,omitempty"`
		FieldCamelCaseName string             `yaml:"field_camel_case_name,omitempty"`
		FieldSnakeCaseName string             `yaml:"field_snake_case_name,omitempty"`
		Documentation      string             `yaml:"documentation,omitempty"`
		Optional           *OptionalType      `yaml:"optional,omitempty"`
		ServiceBucket      *ServiceBucketType `yaml:"service_bucket,omitempty"`
		ServiceObject      *ServiceObjectType `yaml:"service_object,omitempty"`
	}
)

func (jsonField *JsonField) camelCaseName() string {
	if jsonField.FieldCamelCaseName != "" {
		return jsonField.FieldCamelCaseName
	}
	return strcase.ToCamel(jsonField.FieldName)
}

func (jsonStruct *JsonStruct) camelCaseName() string {
	if jsonStruct.CamelCaseName != "" {
		return jsonStruct.CamelCaseName
	}
	return strcase.ToCamel(jsonStruct.Name)
}

func (jsonStruct *JsonStruct) addFields(group *jen.Group, includesJsonTag bool) error {
	for _, field := range jsonStruct.Fields {
		code, err := field.Type.AddTypeToStatement(jen.Id(field.camelCaseName()), field.Optional.ToOptionalType() == OptionalTypeNullable)
		if err != nil {
			return err
		}
		if includesJsonTag {
			jsonTag := field.Key
			switch field.Optional.ToOptionalType() {
			case OptionalTypeOmitEmpty, OptionalTypeNullable:
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
			fieldName := field.camelCaseName()
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

func (jsonStruct *JsonStruct) addGetObjectNameFunc(group *jen.Group, structName string) (bool, error) {
	field := jsonStruct.getServiceObjectField()

	if field == nil || field.ServiceObject.ToServiceObjectType() == ServiceObjectTypeNone {
		return false, nil
	} else if !field.Type.String {
		panic("service object field must be string")
	}

	group.Add(jen.Func().
		Params(jen.Id("j").Op("*").Id(structName)).
		Id("getObjectName").
		Params().
		Params(jen.String()).
		BlockFunc(func(group *jen.Group) {
			fieldName := field.camelCaseName()
			switch field.ServiceObject.ToServiceObjectType() {
			case ServiceObjectTypePlainText:
				if field.Optional.ToOptionalType() == OptionalTypeNullable {
					group.Add(jen.Var().Id("objectName").String())
					group.Add(jen.If(jen.Id("j").Dot(fieldName).Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
						group.Add(jen.Id("objectName").Op("=").Op("*").Id("j").Dot(fieldName))
					}))
					group.Add(jen.Return(jen.Id("objectName")))
				} else {
					group.Return(jen.Id("j").Dot(fieldName))
				}
			case ServiceObjectTypeEntry:
				group.Add(
					jen.Id("parts").
						Op(":=").
						Qual("strings", "SplitN").
						Call(jen.Id("j").Dot(fieldName), jen.Lit(":"), jen.Lit(2)))
				group.Add(jen.If(jen.Len(jen.Id("parts")).Op(">").Lit(1)), jen.BlockFunc(func(group *jen.Group) {
					group.Return(jen.Id("parts").Index(jen.Lit(1)))
				}))
				group.Add(jen.Return(jen.Lit("")))
			default:
				panic("unknown ServiceObjectType")
			}
		}))
	return true, nil
}

func (jsonType *JsonType) generate(group *jen.Group, options CodeGeneratorOptions) error {
	return jsonType.generateType(group, options, true, func() error {
		if jsonType.Any {
			return jsonType.addAnyJsonMarshalerUnmarshaler(group, options.camelCaseName())
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

func (jsonType *JsonType) AddTypeToStatement(statement *jen.Statement, nilable bool) (*jen.Statement, error) {
	if nilable {
		statement = statement.Op("*")
	}
	if jsonType.String {
		return statement.String(), nil
	} else if jsonType.Integer {
		return statement.Int64(), nil
	} else if jsonType.Float {
		return statement.Float64(), nil
	} else if jsonType.Boolean {
		return statement.Bool(), nil
	} else if jsonType.Any {
		return statement.Interface(), nil
	} else if jsonType.StringMap {
		return statement.Map(jen.String()).String(), nil
	} else if jsonType.Array != nil {
		return statement.Id(jsonType.Array.camelCaseName()), nil
	} else if jsonType.Struct != nil {
		return statement.Id(jsonType.Struct.camelCaseName()), nil
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
				group.Return().Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("j").Dot("Body"))
			}),
	)
	return
}

func (jsonArray *JsonArray) camelCaseName() string {
	if jsonArray.CamelCaseName != "" {
		return jsonArray.CamelCaseName
	}
	return strcase.ToCamel(jsonArray.Name)
}

func (jsonArray *JsonArray) addFields(group *jen.Group) error {
	code := jen.Id(jsonArray.camelCaseName()).Id(jsonArray.camelCaseName())
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
	code := jen.Type().Id(jsonArray.camelCaseName())
	if !topLevel {
		code = code.Op("=")
	}
	code = code.Index()
	code, err = jsonArray.Type.AddTypeToStatement(code, false)
	if err != nil {
		return
	}
	group.Add(code)
	if topLevel {
		if err = jsonArray.addJsonMarshalerUnmarshaler(group, options.camelCaseName()); err != nil {
			return
		}
	}

	return
}

func (jsonArray *JsonArray) addJsonMarshalerUnmarshaler(group *jen.Group, structName string) (err error) {
	fieldName := jsonArray.camelCaseName()
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
		if err = field.Type.generateType(group, CodeGeneratorOptions{
			Name:          field.FieldName,
			CamelCaseName: field.FieldCamelCaseName,
			SnakeCaseName: field.FieldSnakeCaseName,
			Documentation: field.Documentation,
		}, false, func() error {
			return nil
		}); err != nil {
			return
		}
	}

	opts := make([]CodeGeneratorOptions, 0, 2)
	if options.camelCaseName() != "" {
		opts = append(opts, options)
	}
	if jsonStruct.camelCaseName() != "" && options.camelCaseName() != jsonStruct.camelCaseName() {
		opts = append(opts, CodeGeneratorOptions{
			Name:          jsonStruct.Name,
			CamelCaseName: jsonStruct.CamelCaseName,
			SnakeCaseName: jsonStruct.SnakeCaseName,
			Documentation: jsonStruct.Documentation,
		})
	}
	if len(opts) == 0 {
		return errors.New("unknown struct name")
	}

	if !topLevel {
		if opts[0].Documentation != "" {
			group.Add(jen.Comment(opts[0].Documentation))
		}
		group.Add(jen.Type().Id(opts[0].camelCaseName()).StructFunc(func(group *jen.Group) {
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
		group.Add(jen.Type().Id(opts[1].camelCaseName()).Op("=").Id(opts[0].camelCaseName()))
	}

	if err = jsonStruct.addJsonMarshalerUnmarshaler(group, opts[0].camelCaseName()); err != nil {
		return
	}
	if err = jsonStruct.generateValidateFunc(group, opts[0].camelCaseName()); err != nil {
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
										fieldName := field.camelCaseName()
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
					fieldName := field.camelCaseName()
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
					fieldName := field.camelCaseName()
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

func (jsonStruct *JsonStruct) getServiceObjectField() *JsonField {
	var serviceObjectField *JsonField = nil

	for i := range jsonStruct.Fields {
		if jsonStruct.Fields[i].ServiceObject.ToServiceObjectType() != ServiceObjectTypeNone {
			if serviceObjectField == nil {
				serviceObjectField = &jsonStruct.Fields[i]
			} else {
				panic("multiple service object fields")
			}
		}
	}
	return serviceObjectField
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
