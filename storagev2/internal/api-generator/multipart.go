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
		Optional      bool                   `yaml:"optional,omitempty"`
	}

	FreeMultipartFormFields struct {
		FieldName     string `yaml:"field_name,omitempty"`
		Documentation string `yaml:"documentation,omitempty"`
	}
)

func (mff *MultipartFormFields) Generate(group *jen.Group, options CodeGeneratorOptions) error {
	var err error
	options.Name = strcase.ToCamel(options.Name)

	group.Add(
		jen.Type().Id(options.Name).StructFunc(func(group *jen.Group) {
			for _, named := range mff.Named {
				code, e := named.Type.AddTypeToStatement(jen.Id("field" + strcase.ToCamel(named.FieldName)))
				if e != nil {
					err = e
					return
				}
				group.Add(code)
				if named.Type.ToMultipartFormDataType() == MultipartFormDataTypeBinaryData {
					group.Add(jen.Id("field" + strcase.ToCamel(named.FieldName) + "_FileName").String())
				}
			}
			if mff.Free != nil {
				group.Add(jen.Id("extendedMap").Map(jen.String()).String())
			}
		}),
	)
	for _, named := range mff.Named {
		if code, err := mff.generateGetterFunc(named, options); err != nil {
			return err
		} else {
			group.Add(code)
		}
		if code, err := mff.generateSetterFunc(named, options); err != nil {
			return err
		} else {
			group.Add(code)
		}
	}
	if code := mff.generateServiceBucketField(options); code != nil {
		group.Add(code)
	}
	if mff.Free != nil {
		group.Add(
			jen.Func().Params(jen.Id("form").Op("*").Id(options.Name)).
				Id("Set").
				Params(jen.Id("key").String(), jen.Id("value").String()).
				Params(jen.Op("*").Id(options.Name)).
				BlockFunc(func(group *jen.Group) {
					group.Add(
						jen.If(jen.Id("form").Dot("extendedMap").Op("==").Nil()).BlockFunc(func(group *jen.Group) {
							group.Add(
								jen.Id("form").Dot("extendedMap").Op("=").Make(jen.Map(jen.String()).String()),
							)
						}),
					)
					group.Add(
						jen.Id("form").Dot("extendedMap").Index(jen.Id("key")).Op("=").Id("value"),
					)
					group.Add(jen.Return(jen.Id("form")))
				}),
		)
	}
	group.Add(
		jen.Func().
			Params(jen.Id("form").Op("*").Id(options.Name)).
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
					field := jen.Id("form").Dot("field" + strcase.ToCamel(named.FieldName))
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
							group.Add(jen.Id("multipartForm").Dot("SetFile").Call(
								jen.Lit(named.Key),
								jen.Id("form").Dot("field"+strcase.ToCamel(named.FieldName)+"_FileName"),
								field,
							))
						}
					})
					if !named.Optional && !named.Type.IsNumeric() {
						code = code.Else().BlockFunc(func(group *jen.Group) {
							group.Add(jen.Return(
								jen.Nil(),
								jen.Qual(PackageNameErrors, "MissingRequiredFieldError").
									ValuesFunc(func(group *jen.Group) {
										group.Add(jen.Id("Name").Op(":").Lit(strcase.ToCamel(named.FieldName)))
									}),
							))
						})
					}
					group.Add(code)
				}
				if mff.Free != nil {
					group.Add(jen.For(jen.List(jen.Id("key"), jen.Id("value")).Op(":=").Range().Id("form").Dot("extendedMap")).BlockFunc(func(group *jen.Group) {
						group.Add(jen.Id("multipartForm").Dot("SetValue").Call(jen.Id("key"), jen.Id("value")))
					}))
				}
				group.Add(jen.Return(jen.Id("multipartForm"), jen.Nil()))
			}),
	)
	return err
}

func (mff *MultipartFormFields) generateGetterFunc(named NamedMultipartFormField, options CodeGeneratorOptions) (jen.Code, error) {
	var (
		fieldName = strcase.ToCamel(named.FieldName)
		err       error
	)
	code := jen.Func().
		Params(jen.Id("form").Op("*").Id(options.Name)).
		Id("Get" + fieldName).
		Params()
	switch named.Type.ToMultipartFormDataType() {
	case MultipartFormDataTypeBinaryData:
		code = code.Params(jen.Qual(PackageNameInternalIo, "ReadSeekCloser"), jen.String()).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.Return(
					jen.Id("form").Dot("field"+fieldName),
					jen.Id("form").Dot("field"+fieldName+"_FileName"),
				))
			})
	default:
		if code, err = named.Type.AddTypeToStatement(code); err != nil {
			return nil, err
		}
		code = code.BlockFunc(func(group *jen.Group) {
			group.Add(jen.Return(jen.Id("form").Dot("field" + fieldName)))
		})
	}
	return code, nil
}

func (mff *MultipartFormFields) generateSetterFunc(named NamedMultipartFormField, options CodeGeneratorOptions) (jen.Code, error) {
	var (
		params    []jen.Code
		fieldName = strcase.ToCamel(named.FieldName)
	)
	switch named.Type.ToMultipartFormDataType() {
	case MultipartFormDataTypeBinaryData:
		params = []jen.Code{jen.Id("value").Qual(PackageNameInternalIo, "ReadSeekCloser"), jen.Id("fileName").String()}
	default:
		p, err := named.Type.AddTypeToStatement(jen.Id("value"))
		if err != nil {
			return nil, err
		}
		params = []jen.Code{p}
	}
	return jen.Func().
		Params(jen.Id("form").Op("*").Id(options.Name)).
		Id("Set" + fieldName).
		Params(params...).
		Params(jen.Op("*").Id(options.Name)).
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Id("form").Dot("field" + fieldName).Op("=").Id("value"))
			if named.Type.ToMultipartFormDataType() == MultipartFormDataTypeBinaryData {
				group.Add(jen.Id("form").Dot("field" + fieldName + "_FileName").Op("=").Id("fileName"))
			}
			group.Add(jen.Return(jen.Id("form")))
		}), nil
}

func (mff *MultipartFormFields) getServiceBucketField() NamedMultipartFormField {
	var serviceBucketField NamedMultipartFormField

	for _, field := range mff.Named {
		if field.ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
			if serviceBucketField.ServiceBucket.ToServiceBucketType() == ServiceBucketTypeNone {
				serviceBucketField = field
			} else {
				panic(fmt.Sprintf("multiple service bucket fields: %s & %s", field.FieldName, serviceBucketField.FieldName))
			}
		}
	}
	return serviceBucketField
}

func (mff *MultipartFormFields) generateServiceBucketField(options CodeGeneratorOptions) jen.Code {
	field := mff.getServiceBucketField()
	if field.ServiceBucket.ToServiceBucketType() == ServiceBucketTypeNone {
		return nil
	}
	return jen.Func().
		Params(jen.Id("form").Op("*").Id(options.Name)).
		Id("getBucketName").
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(jen.String(), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			switch field.ServiceBucket.ToServiceBucketType() {
			case ServiceBucketTypePlainText:
				group.Add(jen.Return(jen.Id("form").Dot("field"+strcase.ToCamel(field.FieldName)), jen.Nil()))
			case ServiceBucketTypeEntry:
				group.Add(
					jen.Return(
						jen.Qual("strings", "SplitN").
							Call(jen.Id("form").Dot("field"+strcase.ToCamel(field.FieldName)), jen.Lit(":"), jen.Lit(2)).
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
						Dot("field" + strcase.ToCamel(field.FieldName)).
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
		})
}
