package main

import (
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
)

type (
	MultipartFormFields struct {
		Named []NamedMultipartFormField `yaml:"named_fields,omitempty"`
		Free  *FreeMultipartFormFields  `yaml:"free_fields,omitempty"`
	}

	NamedMultipartFormField struct {
		FieldName          string                 `yaml:"field_name,omitempty"`
		FieldCamelCaseName string                 `yaml:"field_camel_case_name,omitempty"`
		FieldSnakeCaseName string                 `yaml:"field_snake_case_name,omitempty"`
		Key                string                 `yaml:"key,omitempty"`
		Type               *MultipartFormDataType `yaml:"type,omitempty"`
		Documentation      string                 `yaml:"documentation,omitempty"`
		ServiceBucket      *ServiceBucketType     `yaml:"service_bucket,omitempty"`
		ServiceObject      *ServiceObjectType     `yaml:"service_object,omitempty"`
		Optional           *OptionalType          `yaml:"optional,omitempty"`
	}

	FreeMultipartFormFields struct {
		FieldName          string `yaml:"field_name,omitempty"`
		FieldCamelCaseName string `yaml:"field_camel_case_name,omitempty"`
		FieldSnakeCaseName string `yaml:"field_snake_case_name,omitempty"`
		Documentation      string `yaml:"documentation,omitempty"`
	}
)

func (field *NamedMultipartFormField) camelCaseName() string {
	if field.FieldCamelCaseName != "" {
		return field.FieldCamelCaseName
	}
	return strcase.ToCamel(field.FieldName)
}

func (field *FreeMultipartFormFields) camelCaseName() string {
	if field.FieldCamelCaseName != "" {
		return field.FieldCamelCaseName
	}
	return strcase.ToCamel(field.FieldName)
}

func (mff *MultipartFormFields) addFields(group *jen.Group) error {
	for _, named := range mff.Named {
		fieldName := named.camelCaseName()
		code, err := named.Type.AddTypeToStatement(jen.Id(fieldName), named.Optional.ToOptionalType() == OptionalTypeNullable)
		if err != nil {
			return err
		}
		group.Add(code)
	}
	if free := mff.Free; free != nil {
		group.Add(jen.Id(free.camelCaseName()).Map(jen.String()).String())
	}
	return nil
}

func (mff *MultipartFormFields) addGetBucketNameFunc(group *jen.Group, structName string) (bool, error) {
	field := mff.getServiceBucketField()

	if field == nil || field.ServiceBucket.ToServiceBucketType() == ServiceBucketTypeNone {
		return false, nil
	} else if field.Type.ToMultipartFormDataType() != MultipartFormDataTypeString && field.Type.ToMultipartFormDataType() != MultipartFormDataTypeUploadToken {
		panic(fmt.Sprintf("service bucket field must be string: %s", field.FieldName))
	}

	group.Add(jen.Func().
		Params(jen.Id("form").Op("*").Id(structName)).
		Id("getBucketName").
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(jen.String(), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			fieldName := field.camelCaseName()
			switch field.ServiceBucket.ToServiceBucketType() {
			case ServiceBucketTypePlainText:
				group.Add(jen.Return(jen.Id("form").Dot(fieldName), jen.Nil()))
			case ServiceBucketTypeEntry:
				group.Add(
					jen.Return(
						jen.Qual("strings", "SplitN").
							Call(jen.Id("form").Dot(fieldName), jen.Lit(":"), jen.Lit(2)).
							Index(jen.Lit(0)),
						jen.Nil(),
					),
				)
			case ServiceBucketTypeUploadToken:
				group.Add(
					jen.Id("putPolicy").
						Op(",").
						Err().
						Op(":=").
						Id("form").
						Dot(fieldName).
						Dot("GetPutPolicy").
						Call(jen.Id("ctx")),
				)
				group.Add(
					jen.If(jen.Err().Op("!=").Nil()).
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

func (mff *MultipartFormFields) addGetObjectNameFunc(group *jen.Group, structName string) (bool, error) {
	field := mff.getServiceObjectField()

	if field == nil || field.ServiceObject.ToServiceObjectType() == ServiceObjectTypeNone {
		return false, nil
	} else if field.Type.ToMultipartFormDataType() != MultipartFormDataTypeString {
		panic(fmt.Sprintf("service object field must be string: %s", field.FieldName))
	}

	group.Add(jen.Func().
		Params(jen.Id("form").Op("*").Id(structName)).
		Id("getObjectName").
		Params().
		Params(jen.String()).
		BlockFunc(func(group *jen.Group) {
			fieldName := field.camelCaseName()
			switch field.ServiceObject.ToServiceObjectType() {
			case ServiceObjectTypePlainText:
				if field.Optional.ToOptionalType() == OptionalTypeNullable {
					group.Add(jen.Var().Id("objectName").String())
					group.Add(jen.If(jen.Id("form").Dot(fieldName).Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
						group.Add(jen.Id("objectName").Op("=").Op("*").Id("form").Dot(fieldName))
					}))
					group.Add(jen.Return(jen.Id("objectName")))
				} else {
					group.Return(jen.Id("form").Dot(fieldName))
				}
			case ServiceObjectTypeEntry:
				group.Add(
					jen.Id("parts").
						Op(":=").
						Qual("strings", "SplitN").
						Call(jen.Id("form").Dot(fieldName), jen.Lit(":"), jen.Lit(2)))
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

func (mff *MultipartFormFields) addBuildFunc(group *jen.Group, structName string) error {
	var err error
	group.Add(
		jen.Func().
			Params(jen.Id("form").Op("*").Id(structName)).
			Id("build").
			Params(jen.Id("ctx").Qual("context", "Context")).
			Params(jen.Op("*").Qual(PackageNameHTTPClient, "MultipartForm"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.Id("multipartForm").
						Op(":=").
						New(jen.Qual(PackageNameHTTPClient, "MultipartForm")),
				)
				for _, named := range mff.Named {
					var zeroValue interface{}
					if named.Optional.ToOptionalType() != OptionalTypeNullable {
						zeroValue, err = named.Type.ZeroValue()
					}
					if err != nil {
						return
					}
					fieldName := named.camelCaseName()
					field := jen.Id("form").Dot(fieldName)
					var cond *jen.Statement
					if named.Type.ToMultipartFormDataType() == MultipartFormDataTypeBinaryData {
						cond = field.Clone().Dot("Data").Op("!=").Nil()
					} else if zeroValue == nil {
						cond = field.Clone().Op("!=").Nil()
					} else {
						cond = field.Clone().Op("!=").Lit(zeroValue)
					}
					if named.Optional.ToOptionalType() == OptionalTypeNullable {
						field = jen.Op("*").Add(field)
					}
					code := jen.If(cond).BlockFunc(func(group *jen.Group) {
						switch named.Type.ToMultipartFormDataType() {
						case MultipartFormDataTypeString:
							group.Add(jen.Id("multipartForm").Dot("SetValue").Call(
								jen.Lit(named.Key),
								field,
							))
						case MultipartFormDataTypeInteger:
							group.Add(jen.Id("multipartForm").Dot("SetValue").Call(
								jen.Lit(named.Key),
								jen.Qual("strconv", "FormatInt").Call(field, jen.Lit(10)),
							))
						case MultipartFormDataTypeUploadToken:
							group.Add(
								jen.Id("upToken").
									Op(",").
									Err().
									Op(":=").
									Add(field).
									Dot("GetUpToken").
									Call(jen.Id("ctx")),
							)
							group.Add(
								jen.If(jen.Err().Op("!=").Nil()).
									BlockFunc(func(group *jen.Group) {
										group.Add(jen.Return(jen.Nil(), jen.Err()))
									}),
							)
							group.Add(jen.Id("multipartForm").Dot("SetValue").Call(
								jen.Lit(named.Key),
								jen.Id("upToken"),
							))
						case MultipartFormDataTypeBinaryData:
							group.Add(jen.If(jen.Id("form").Dot(fieldName).Dot("Name").Op("==").Lit("").BlockFunc(func(group *jen.Group) {
								group.Add(jen.Return(
									jen.Nil(),
									jen.Qual(PackageNameErrors, "MissingRequiredFieldError").
										ValuesFunc(func(group *jen.Group) {
											group.Add(jen.Id("Name").Op(":").Lit(fieldName + ".Name"))
										}),
								))
							})))
							group.Add(jen.Id("multipartForm").Dot("SetFile").Call(
								jen.Lit(named.Key),
								jen.Id("form").Dot(fieldName).Dot("Name"),
								jen.Id("form").Dot(fieldName).Dot("ContentType"),
								jen.Id("form").Dot(fieldName).Dot("Data"),
							))
						}
					})
					if named.Optional.ToOptionalType() == OptionalTypeRequired {
						code = code.Else().BlockFunc(func(group *jen.Group) {
							group.Add(jen.Return(
								jen.Nil(),
								jen.Qual(PackageNameErrors, "MissingRequiredFieldError").
									ValuesFunc(func(group *jen.Group) {
										group.Add(jen.Id("Name").Op(":").Lit(fieldName))
									}),
							))
						})
					}
					group.Add(code)
				}
				if free := mff.Free; free != nil {
					group.Add(jen.For(jen.List(jen.Id("key"), jen.Id("value")).Op(":=").Range().Id("form").Dot(free.camelCaseName())).BlockFunc(func(group *jen.Group) {
						group.Add(jen.Id("multipartForm").Dot("SetValue").Call(jen.Id("key"), jen.Id("value")))
					}))
				}
				group.Add(jen.Return(jen.Id("multipartForm"), jen.Nil()))
			}),
	)
	return err
}

func (mff *MultipartFormFields) getServiceBucketField() *NamedMultipartFormField {
	var serviceBucketField *NamedMultipartFormField

	for i := range mff.Named {
		if mff.Named[i].ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
			if serviceBucketField == nil {
				serviceBucketField = &mff.Named[i]
			} else {
				panic(fmt.Sprintf("multiple service bucket fields: %s & %s", mff.Named[i].FieldName, serviceBucketField.FieldName))
			}
		}
	}
	return serviceBucketField
}

func (mff *MultipartFormFields) getServiceObjectField() *NamedMultipartFormField {
	var serviceObjectField *NamedMultipartFormField

	for i := range mff.Named {
		if mff.Named[i].ServiceObject.ToServiceObjectType() != ServiceObjectTypeNone {
			if serviceObjectField == nil {
				serviceObjectField = &mff.Named[i]
			} else {
				panic(fmt.Sprintf("multiple service object fields: %s & %s", mff.Named[i].FieldName, serviceObjectField.FieldName))
			}
		}
	}
	return serviceObjectField
}
