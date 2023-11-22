package main

import (
	"errors"
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
)

type (
	PathParams struct {
		Named []NamedPathParam `yaml:"named,omitempty"`
		Free  *FreePathParams  `yaml:"free,omitempty"`
	}

	NamedPathParam struct {
		PathSegment   string             `yaml:"path_segment,omitempty"`
		FieldName     string             `yaml:"field_name,omitempty"`
		Type          *StringLikeType    `yaml:"type,omitempty"`
		Documentation string             `yaml:"documentation,omitempty"`
		Encode        *EncodeType        `yaml:"encode,omitempty"`
		ServiceBucket *ServiceBucketType `yaml:"service_bucket,omitempty"`
		Optional      *OptionalType      `yaml:"optional,omitempty"`
	}

	FreePathParams struct {
		FieldName        string      `yaml:"field_name,omitempty"`
		Documentation    string      `yaml:"documentation,omitempty"`
		EncodeParamKey   *EncodeType `yaml:"encode_param_key"`
		EncodeParamValue *EncodeType `yaml:"encode_param_value"`
	}
)

func (pp *PathParams) Generate(group *jen.Group, options CodeGeneratorOptions) error {
	var err error
	options.Name = strcase.ToCamel(options.Name)
	if options.Documentation != "" {
		group.Add(jen.Comment(options.Documentation))
	}
	group.Add(
		jen.Type().Id(options.Name).StructFunc(func(group *jen.Group) {
			for _, namedPathParam := range pp.Named {
				code, e := namedPathParam.Type.AddTypeToStatement(jen.Id("field" + strcase.ToCamel(namedPathParam.FieldName)))
				if e != nil {
					err = e
					return
				}
				group.Add(code)
			}
			if pp.Free != nil {
				group.Add(jen.Id("extendedSegments").Index().Add(jen.String()))
			}
		}),
	)
	for _, named := range pp.Named {
		if err = pp.generateGetterFunc(group, named, options); err != nil {
			return err
		}
		if err = pp.generateSetterFunc(group, named, options); err != nil {
			return err
		}
	}
	if code := pp.generateServiceBucketField(options); code != nil {
		group.Add(code)
	}
	if pp.Free != nil {
		group.Add(jen.Comment("追加自定义路径参数"))
		group.Add(
			jen.Func().
				Params(jen.Id("path").Op("*").Id(options.Name)).
				Id("Append").
				Params(jen.Id("key").String(), jen.Id("value").String()).
				Params(jen.Op("*").Id(options.Name)).
				BlockFunc(func(group *jen.Group) {
					var keyCode, valueCode jen.Code
					switch pp.Free.EncodeParamKey.ToEncodeType() {
					case EncodeTypeNone:
						keyCode = jen.Id("key")
					case EncodeTypeUrlsafeBase64, EncodeTypeUrlsafeBase64OrNone:
						keyCode = jen.Qual("encoding/base64", "URLEncoding").
							Dot("EncodeToString").
							Call(jen.Index().Byte().Parens(jen.Id("key")))
					}
					group.Add(
						jen.Id("path").Dot("extendedSegments").Op("=").Append(
							jen.Id("path").Dot("extendedSegments"),
							keyCode,
						),
					)
					switch pp.Free.EncodeParamValue.ToEncodeType() {
					case EncodeTypeNone:
						valueCode = jen.Id("value")
					case EncodeTypeUrlsafeBase64, EncodeTypeUrlsafeBase64OrNone:
						valueCode = jen.Qual("encoding/base64", "URLEncoding").
							Dot("EncodeToString").
							Call(jen.Index().Byte().Parens(jen.Id("value")))
					}
					group.Add(
						jen.Id("path").Dot("extendedSegments").Op("=").Append(
							jen.Id("path").Dot("extendedSegments"),
							valueCode,
						),
					)
					group.Add(jen.Return(jen.Id("path")))
				}),
		)
	}
	group.Add(
		jen.Func().
			Params(jen.Id("path").Op("*").Id(options.Name)).
			Id("build").
			Params().
			Params(jen.Index().Add(jen.String()), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.Var().Id("allSegments").Index().Add(jen.String()))
				for _, namedPathParam := range pp.Named {
					var (
						code      jen.Code
						isNone    bool
						fieldName = strcase.ToCamel(namedPathParam.FieldName)
						field     = jen.Id("path").Dot("field" + fieldName)
					)
					switch namedPathParam.Type.ToStringLikeType() {
					case StringLikeTypeString:
						switch namedPathParam.Encode.ToEncodeType() {
						case EncodeTypeNone:
							code = field.Clone()
						case EncodeTypeUrlsafeBase64, EncodeTypeUrlsafeBase64OrNone:
							code = jen.Qual("encoding/base64", "URLEncoding").Dot("EncodeToString").Call(jen.Index().Byte().Parens(field.Clone()))
							if namedPathParam.Encode.ToEncodeType() == EncodeTypeUrlsafeBase64OrNone {
								isNone = true
							}
						}
					case StringLikeTypeInteger, StringLikeTypeFloat, StringLikeTypeBoolean:
						code, _ = namedPathParam.Type.GenerateConvertCodeToString(field)
					default:
						err = errors.New("unknown type")
						return
					}
					zeroValue, e := namedPathParam.Type.ZeroValue()
					if e != nil {
						err = e
						return
					}
					condition := field.Clone()
					if v, ok := zeroValue.(bool); !ok || v {
						condition = condition.Op("!=").Lit(zeroValue)
					}
					appendPathSegment := func(pathSegment string, value jen.Code) func(group *jen.Group) {
						return func(group *jen.Group) {
							codes := []jen.Code{jen.Id("allSegments")}
							if pathSegment != "" {
								codes = append(codes, jen.Lit(pathSegment))
							}
							codes = append(codes, value)
							group.Add(
								jen.Id("allSegments").Op("=").Append(codes...),
							)
						}
					}
					appendMissingRequiredFieldErrorFunc := func(fieldName string) func(group *jen.Group) {
						return func(group *jen.Group) {
							group.Add(jen.Return(
								jen.Nil(),
								jen.Qual(PackageNameErrors, "MissingRequiredFieldError").
									ValuesFunc(func(group *jen.Group) {
										group.Add(jen.Id("Name").Op(":").Lit(fieldName))
									}),
							))
						}
					}

					if isNone {
						group.Add(
							jen.If(condition).
								BlockFunc(appendPathSegment(namedPathParam.PathSegment, code)).
								Else().
								BlockFunc(appendPathSegment(namedPathParam.PathSegment, jen.Lit("~"))),
						)
					} else {
						switch namedPathParam.Optional.ToOptionalType() {
						case OptionalTypeRequired:
							group.Add(
								jen.If(condition).
									BlockFunc(appendPathSegment(namedPathParam.PathSegment, code)).
									Else().
									BlockFunc(appendMissingRequiredFieldErrorFunc(fieldName)),
							)
						case OptionalTypeOmitEmpty:
							group.Add(
								jen.If(condition).
									BlockFunc(appendPathSegment(namedPathParam.PathSegment, code)),
							)
						case OptionalTypeKeepEmpty:
							appendPathSegment(namedPathParam.PathSegment, code)(group)
						}
					}
				}
				if pp.Free != nil {
					group.Add(jen.Id("allSegments").Op("=").Append(jen.Id("allSegments"), jen.Id("path").Dot("extendedSegments").Op("...")))
				}
				group.Add(jen.Return(jen.Id("allSegments"), jen.Nil()))
			}),
	)
	return err
}

func (pp *PathParams) GenerateAliasesFor(group *jen.Group, structName, fieldName string) error {
	for _, named := range pp.Named {
		if err := pp.generateAliasGetterFunc(group, named, structName, fieldName); err != nil {
			return err
		}
		if err := pp.generateAliasSetterFunc(group, named, structName, fieldName); err != nil {
			return err
		}
	}
	return nil
}

func (pp *PathParams) generateAliasGetterFunc(group *jen.Group, named NamedPathParam, structName, fieldName string) error {
	if named.Documentation != "" {
		group.Add(jen.Comment(named.Documentation))
	}
	code := jen.Func().
		Params(jen.Id("request").Op("*").Id(structName)).
		Id(makeGetterMethodName(named.FieldName)).
		Params()
	code, err := named.Type.AddTypeToStatement(code)
	if err != nil {
		return err
	}
	code = code.BlockFunc(func(group *jen.Group) {
		group.Add(jen.Return(jen.Id("request").Dot(fieldName).Dot(makeGetterMethodName(named.FieldName)).Call()))
	})
	group.Add(code)
	return nil
}

func (pp *PathParams) generateAliasSetterFunc(group *jen.Group, named NamedPathParam, structName, fieldName string) error {
	if named.Documentation != "" {
		group.Add(jen.Comment(named.Documentation))
	}
	params, err := named.Type.AddTypeToStatement(jen.Id("value"))
	if err != nil {
		return err
	}
	group.Add(jen.Func().
		Params(jen.Id("request").Op("*").Id(structName)).
		Id(makeSetterMethodName(named.FieldName)).
		Params(params).
		Params(jen.Op("*").Id(structName)).
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Id("request").Dot(fieldName).Dot(makeSetterMethodName(named.FieldName)).Call(jen.Id("value")))
			group.Add(jen.Return(jen.Id("request")))
		}))
	return nil
}

func (pp *PathParams) generateGetterFunc(group *jen.Group, named NamedPathParam, options CodeGeneratorOptions) error {
	if named.Documentation != "" {
		group.Add(jen.Comment(named.Documentation))
	}
	code := jen.Func().
		Params(jen.Id("pp").Op("*").Id(options.Name)).
		Id(makeGetterMethodName(named.FieldName)).
		Params()
	code, err := named.Type.AddTypeToStatement(code)
	if err != nil {
		return err
	}
	code = code.BlockFunc(func(group *jen.Group) {
		group.Add(jen.Return(jen.Id("pp").Dot("field" + strcase.ToCamel(named.FieldName))))
	})
	group.Add(code)
	return nil
}

func (pp *PathParams) generateSetterFunc(group *jen.Group, named NamedPathParam, options CodeGeneratorOptions) error {
	if named.Documentation != "" {
		group.Add(jen.Comment(named.Documentation))
	}
	params, err := named.Type.AddTypeToStatement(jen.Id("value"))
	if err != nil {
		return err
	}
	group.Add(jen.Func().
		Params(jen.Id("pp").Op("*").Id(options.Name)).
		Id(makeSetterMethodName(named.FieldName)).
		Params(params).
		Params(jen.Op("*").Id(options.Name)).
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Id("pp").Dot("field" + strcase.ToCamel(named.FieldName)).Op("=").Id("value"))
			group.Add(jen.Return(jen.Id("pp")))
		}))
	return nil
}

func (pp *PathParams) getServiceBucketField() *NamedPathParam {
	var serviceBucketField *NamedPathParam

	for i := range pp.Named {
		if pp.Named[i].ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
			if serviceBucketField == nil {
				serviceBucketField = &pp.Named[i]
			} else {
				panic(fmt.Sprintf("multiple service bucket fields: %s & %s", pp.Named[i].FieldName, serviceBucketField.FieldName))
			}
		}
	}
	return serviceBucketField
}

func (pp *PathParams) generateServiceBucketField(options CodeGeneratorOptions) jen.Code {
	field := pp.getServiceBucketField()
	if field == nil || field.ServiceBucket.ToServiceBucketType() == ServiceBucketTypeNone {
		return nil
	} else if field.Type.ToStringLikeType() != StringLikeTypeString {
		panic(fmt.Sprintf("service bucket field must be string: %s", field.FieldName))
	}
	return jen.Func().
		Params(jen.Id("pp").Op("*").Id(options.Name)).
		Id("getBucketName").
		Params().
		Params(jen.String(), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			switch field.ServiceBucket.ToServiceBucketType() {
			case ServiceBucketTypePlainText:
				group.Add(jen.Return(jen.Id("pp").Dot("field"+strcase.ToCamel(field.FieldName)), jen.Nil()))
			case ServiceBucketTypeEntry:
				group.Add(
					jen.Return(
						jen.Qual("strings", "SplitN").
							Call(jen.Id("pp").Dot("field"+strcase.ToCamel(field.FieldName)), jen.Lit(":"), jen.Lit(2)).
							Index(jen.Lit(0)),
						jen.Nil(),
					),
				)
			case ServiceBucketTypeUploadToken:
				group.Add(
					jen.If(jen.Id("putPolicy"), jen.Err()).
						Op(":=").
						Qual(PackageNameUpToken, "NewParser").
						Call(jen.Id("pp").Dot("field" + strcase.ToCamel(field.FieldName))).
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
