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
		PathSegment        string             `yaml:"path_segment,omitempty"`
		FieldName          string             `yaml:"field_name,omitempty"`
		FieldCamelCaseName string             `yaml:"field_camel_case_name,omitempty"`
		FieldSnakeCaseName string             `yaml:"field_snake_case_name,omitempty"`
		Type               *StringLikeType    `yaml:"type,omitempty"`
		Documentation      string             `yaml:"documentation,omitempty"`
		Encode             *EncodeType        `yaml:"encode,omitempty"`
		ServiceBucket      *ServiceBucketType `yaml:"service_bucket,omitempty"`
		ServiceObject      *ServiceObjectType `yaml:"service_object,omitempty"`
		Optional           *OptionalType      `yaml:"optional,omitempty"`
	}

	FreePathParams struct {
		FieldName          string      `yaml:"field_name,omitempty"`
		FieldCamelCaseName string      `yaml:"field_camel_case_name,omitempty"`
		FieldSnakeCaseName string      `yaml:"field_snake_case_name,omitempty"`
		Documentation      string      `yaml:"documentation,omitempty"`
		EncodeParamKey     *EncodeType `yaml:"encode_param_key"`
		EncodeParamValue   *EncodeType `yaml:"encode_param_value"`
	}
)

func (pp *NamedPathParam) camelCaseName() string {
	if pp.FieldCamelCaseName != "" {
		return pp.FieldCamelCaseName
	}
	return strcase.ToCamel(pp.FieldName)
}

func (fpp *FreePathParams) camelCaseName() string {
	if fpp.FieldCamelCaseName != "" {
		return fpp.FieldCamelCaseName
	}
	return strcase.ToCamel(fpp.FieldName)
}

func (pp *PathParams) addFields(group *jen.Group) error {
	for _, namedPathParam := range pp.Named {
		nilable := namedPathParam.Encode.ToEncodeType() == EncodeTypeUrlsafeBase64OrNone || namedPathParam.Optional.ToOptionalType() == OptionalTypeNullable
		code, err := namedPathParam.Type.AddTypeToStatement(jen.Id(namedPathParam.camelCaseName()), nilable)
		if err != nil {
			return err
		}
		if namedPathParam.Documentation != "" {
			code = code.Comment(namedPathParam.Documentation)
		}
		group.Add(code)
	}
	if free := pp.Free; free != nil {
		freeFieldName := free.camelCaseName()
		code := jen.Id(freeFieldName).Map(jen.String()).String()
		if free.Documentation != "" {
			code = code.Comment(free.Documentation)
		}
		group.Add(code)
	}
	return nil
}

func (pp *PathParams) addGetBucketNameFunc(group *jen.Group, structName string) (bool, error) {
	field := pp.getServiceBucketField()
	if field == nil || field.ServiceBucket.ToServiceBucketType() == ServiceBucketTypeNone {
		return false, nil
	} else if field.Type.ToStringLikeType() != StringLikeTypeString {
		panic(fmt.Sprintf("service bucket field must be string: %s", field.FieldName))
	}
	group.Add(jen.Func().
		Params(jen.Id("pp").Op("*").Id(structName)).
		Id("getBucketName").
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(jen.String(), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			fieldName := field.camelCaseName()
			switch field.ServiceBucket.ToServiceBucketType() {
			case ServiceBucketTypePlainText:
				group.Add(jen.Return(jen.Id("pp").Dot(fieldName), jen.Nil()))
			case ServiceBucketTypeEntry:
				group.Add(
					jen.Return(
						jen.Qual("strings", "SplitN").
							Call(jen.Id("pp").Dot(fieldName), jen.Lit(":"), jen.Lit(2)).
							Index(jen.Lit(0)),
						jen.Nil(),
					),
				)
			case ServiceBucketTypeUploadToken:
				group.Add(
					jen.If(jen.Id("putPolicy"), jen.Err()).
						Op(":=").
						Qual(PackageNameUpToken, "NewParser").
						Call(jen.Id("pp").Dot(fieldName)).
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

func (pp *PathParams) addGetObjectNameFunc(group *jen.Group, structName string) (bool, error) {
	field := pp.getServiceObjectField()
	if field == nil || field.ServiceObject.ToServiceObjectType() == ServiceObjectTypeNone {
		return false, nil
	} else if field.Type.ToStringLikeType() != StringLikeTypeString {
		panic(fmt.Sprintf("service object field must be string: %s", field.FieldName))
	}
	group.Add(jen.Func().
		Params(jen.Id("pp").Op("*").Id(structName)).
		Id("getObjectName").
		Params().
		Params(jen.String()).
		BlockFunc(func(group *jen.Group) {
			fieldName := field.camelCaseName()
			switch field.ServiceObject.ToServiceObjectType() {
			case ServiceObjectTypePlainText:
				if field.Optional.ToOptionalType() == OptionalTypeNullable {
					group.Add(jen.Var().Id("objectName").String())
					group.Add(jen.If(jen.Id("pp").Dot(fieldName).Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
						group.Add(jen.Id("objectName").Op("=").Op("*").Id("pp").Dot(fieldName))
					}))
					group.Add(jen.Return(jen.Id("objectName")))
				} else {
					group.Return(jen.Id("pp").Dot(fieldName))
				}
			case ServiceObjectTypeEntry:
				group.Add(
					jen.Id("parts").
						Op(":=").
						Qual("strings", "SplitN").
						Call(jen.Id("pp").Dot(fieldName), jen.Lit(":"), jen.Lit(2)))
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

func (pp *PathParams) addBuildFunc(group *jen.Group, structName string) error {
	var err error
	group.Add(
		jen.Func().
			Params(jen.Id("path").Op("*").Id(structName)).
			Id("buildPath").
			Params().
			Params(jen.Index().Add(jen.String()), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				guessPathSegmentsCount := 0
				for _, namedPathParam := range pp.Named {
					guessPathSegmentsCount += 1
					if namedPathParam.PathSegment != "" {
						guessPathSegmentsCount += 1
					}
				}
				group.Add(jen.Id("allSegments").Op(":=").Make(jen.Index().Add(jen.String()), jen.Lit(0), jen.Lit(guessPathSegmentsCount)))
				for _, namedPathParam := range pp.Named {
					var (
						code                jen.Code
						urlSafeBase64IsNone = namedPathParam.Encode.ToEncodeType() == EncodeTypeUrlsafeBase64OrNone
						nilable             = urlSafeBase64IsNone || namedPathParam.Optional.ToOptionalType() == OptionalTypeNullable
						fieldName           = namedPathParam.camelCaseName()
						field               = jen.Id("path").Dot(fieldName)
						unreferencedField   = field.Clone()
					)
					if nilable {
						unreferencedField = jen.Op("*").Add(unreferencedField)
					}
					switch namedPathParam.Type.ToStringLikeType() {
					case StringLikeTypeString:
						switch namedPathParam.Encode.ToEncodeType() {
						case EncodeTypeNone:
							code = unreferencedField.Clone()
						case EncodeTypeUrlsafeBase64, EncodeTypeUrlsafeBase64OrNone:
							code = jen.Qual("encoding/base64", "URLEncoding").Dot("EncodeToString").Call(jen.Index().Byte().Parens(unreferencedField.Clone()))
						}
					case StringLikeTypeInteger, StringLikeTypeFloat, StringLikeTypeBoolean:
						code, _ = namedPathParam.Type.GenerateConvertCodeToString(unreferencedField.Clone())
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
					if nilable {
						condition = condition.Op("!=").Nil()
					} else if v, ok := zeroValue.(bool); !ok || v {
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

					if urlSafeBase64IsNone {
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
						case OptionalTypeOmitEmpty, OptionalTypeNullable:
							group.Add(
								jen.If(condition).
									BlockFunc(appendPathSegment(namedPathParam.PathSegment, code)),
							)
						case OptionalTypeKeepEmpty:
							appendPathSegment(namedPathParam.PathSegment, code)(group)
						}
					}
				}
				if free := pp.Free; free != nil {
					freeFieldName := free.camelCaseName()
					group.Add(
						jen.For(
							jen.Id("key").
								Op(",").
								Id("value").
								Op(":=").
								Range().
								Id("path").
								Dot(freeFieldName)).
							BlockFunc(func(group *jen.Group) {
								var keyCode, valueCode jen.Code
								switch free.EncodeParamKey.ToEncodeType() {
								case EncodeTypeNone:
									keyCode = jen.Id("key")
								case EncodeTypeUrlsafeBase64, EncodeTypeUrlsafeBase64OrNone:
									keyCode = jen.Qual("encoding/base64", "URLEncoding").
										Dot("EncodeToString").
										Call(jen.Index().Byte().Parens(jen.Id("key")))
								}
								group.Add(
									jen.Id("allSegments").Op("=").Append(jen.Id("allSegments"), keyCode),
								)
								switch free.EncodeParamValue.ToEncodeType() {
								case EncodeTypeNone:
									valueCode = jen.Id("value")
								case EncodeTypeUrlsafeBase64, EncodeTypeUrlsafeBase64OrNone:
									valueCode = jen.Qual("encoding/base64", "URLEncoding").
										Dot("EncodeToString").
										Call(jen.Index().Byte().Parens(jen.Id("value")))
								}
								group.Add(
									jen.Id("allSegments").Op("=").Append(jen.Id("allSegments"), valueCode),
								)
							}),
					)
				}
				group.Add(jen.Return(jen.Id("allSegments"), jen.Nil()))
			}),
	)
	return err
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

func (pp *PathParams) getServiceObjectField() *NamedPathParam {
	var serviceObjectField *NamedPathParam

	for i := range pp.Named {
		if pp.Named[i].ServiceObject.ToServiceObjectType() != ServiceObjectTypeNone {
			if serviceObjectField == nil {
				serviceObjectField = &pp.Named[i]
			} else {
				panic(fmt.Sprintf("multiple service object fields: %s & %s", pp.Named[i].FieldName, serviceObjectField.FieldName))
			}
		}
	}
	return serviceObjectField
}
