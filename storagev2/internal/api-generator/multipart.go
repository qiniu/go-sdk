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
		FieldName     string                 `yaml:"field_name,omitempty"`
		Key           string                 `yaml:"key,omitempty"`
		Type          *MultipartFormDataType `yaml:"type,omitempty"`
		Documentation string                 `yaml:"documentation,omitempty"`
		ServiceBucket *ServiceBucketType     `yaml:"service_bucket,omitempty"`
		Optional      *OptionalType          `yaml:"optional,omitempty"`
	}

	FreeMultipartFormFields struct {
		FieldName     string `yaml:"field_name,omitempty"`
		Documentation string `yaml:"documentation,omitempty"`
	}
)

func (mff *MultipartFormFields) addFields(group *jen.Group) error {
	for _, named := range mff.Named {
		fieldName := strcase.ToCamel(named.FieldName)
		code, err := named.Type.AddTypeToStatement(jen.Id(fieldName))
		if err != nil {
			return err
		}
		group.Add(code)
		if named.Type.ToMultipartFormDataType() == MultipartFormDataTypeBinaryData {
			group.Add(jen.Id(fieldName + "_FileName").String())
		}
	}
	if free := mff.Free; free != nil {
		group.Add(jen.Id(strcase.ToCamel(free.FieldName)).Map(jen.String()).String())
	}
	return nil
}

func (mff *MultipartFormFields) addGetBucketNameFunc(group *jen.Group, structName string) (bool, error) {
	field := mff.getServiceBucketField()

	if field == nil || field.ServiceBucket.ToServiceBucketType() == ServiceBucketTypeNone {
		return false, nil
	}

	group.Add(jen.Func().
		Params(jen.Id("form").Op("*").Id(structName)).
		Id("getBucketName").
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(jen.String(), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			fieldName := strcase.ToCamel(field.FieldName)
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
						Dot("RetrievePutPolicy").
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

func (mff *MultipartFormFields) addBuildFunc(group *jen.Group, structName string) error {
	var err error
	group.Add(
		jen.Func().
			Params(jen.Id("form").Op("*").Id(structName)).
			Id("build").
			Params(jen.Id("ctx").Qual("context", "Context")).
			Params(jen.Op("*").Qual(PackageNameHttpClient, "MultipartForm"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.Id("multipartForm").
						Op(":=").
						New(jen.Qual(PackageNameHttpClient, "MultipartForm")),
				)
				for _, named := range mff.Named {
					zeroValue, e := named.Type.ZeroValue()
					if e != nil {
						err = e
						return
					}
					fieldName := strcase.ToCamel(named.FieldName)
					field := jen.Id("form").Dot(fieldName)
					var cond *jen.Statement
					if zeroValue == nil {
						cond = field.Clone().Op("!=").Nil()
					} else {
						cond = field.Clone().Op("!=").Lit(zeroValue)
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
									Dot("RetrieveUpToken").
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
							group.Add(jen.If(jen.Id("form").Dot(fieldName + "_FileName").Op("==").Lit("").BlockFunc(func(group *jen.Group) {
								group.Add(jen.Return(
									jen.Nil(),
									jen.Qual(PackageNameErrors, "MissingRequiredFieldError").
										ValuesFunc(func(group *jen.Group) {
											group.Add(jen.Id("Name").Op(":").Lit(fieldName + "_FileName"))
										}),
								))
							})))
							group.Add(jen.Id("multipartForm").Dot("SetFile").Call(
								jen.Lit(named.Key),
								jen.Id("form").Dot(fieldName+"_FileName"),
								field,
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
					group.Add(jen.For(jen.List(jen.Id("key"), jen.Id("value")).Op(":=").Range().Id("form").Dot(strcase.ToCamel(free.FieldName))).BlockFunc(func(group *jen.Group) {
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
