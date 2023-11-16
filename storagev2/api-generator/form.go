package main

import (
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
)

type (
	FormUrlencodedRequestStruct struct {
		Fields []FormUrlencodedRequestField `yaml:"fields,omitempty"`
	}

	FormUrlencodedRequestField struct {
		FieldName     string             `yaml:"field_name,omitempty"`
		Key           string             `yaml:"key,omitempty"`
		Documentation string             `yaml:"documentation,omitempty"`
		Type          *StringLikeType    `yaml:"type,omitempty"`
		Multiple      bool               `yaml:"multiple,omitempty"`
		ServiceBucket *ServiceBucketType `yaml:"service_bucket,omitempty"`
	}
)

func (form *FormUrlencodedRequestStruct) Generate(group *jen.Group, options CodeGeneratorOptions) error {
	var err error
	options.Name = strcase.ToCamel(options.Name)
	group.Add(
		jen.Type().Id(options.Name).StructFunc(func(group *jen.Group) {
			for _, field := range form.Fields {
				if code, e := form.generateField(field); e != nil {
					err = e
					return
				} else {
					group.Add(code)
				}
			}
		}),
	)
	if err != nil {
		return err
	}

	for _, field := range form.Fields {
		if code, err := form.generateGetterFunc(field, options); err != nil {
			return err
		} else {
			group.Add(code)
		}
		if code, err := form.generateSetterFunc(field, options); err != nil {
			return err
		} else {
			group.Add(code)
		}
	}

	if code := form.generateServiceBucketField(options); code != nil {
		group.Add(code)
	}

	group.Add(
		jen.Func().
			Params(jen.Id("form").Op("*").Id(options.Name)).
			Id("build").
			Params().
			Params(jen.Qual("net/url", "Values"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.Id("formValues").Op(":=").Make(jen.Qual("net/url", "Values")),
				)
				for _, field := range form.Fields {
					if code, e := form.generateSetCall(field); e != nil {
						err = e
						return
					} else {
						group.Add(code)
					}
				}
				group.Add(jen.Return(jen.Id("formValues"), jen.Nil()))
			}),
	)

	return err
}

func (form *FormUrlencodedRequestStruct) generateField(field FormUrlencodedRequestField) (jen.Code, error) {
	code := jen.Id("field" + strcase.ToCamel(field.FieldName))
	if field.Multiple {
		code = code.Index()
	}
	code, err := field.Type.AddTypeToStatement(code)
	if err != nil {
		return nil, err
	}
	if field.Documentation != "" {
		code = code.Comment(field.Documentation)
	}
	return code, nil
}

func (form *FormUrlencodedRequestStruct) generateGetterFunc(field FormUrlencodedRequestField, options CodeGeneratorOptions) (jen.Code, error) {
	code := jen.Func().
		Params(jen.Id("form").Op("*").Id(options.Name)).
		Id("Get" + strcase.ToCamel(field.FieldName)).
		Params()
	if field.Multiple {
		code = code.Index()
	}
	code, err := field.Type.AddTypeToStatement(code)
	if err != nil {
		return nil, err
	}
	code = code.BlockFunc(func(group *jen.Group) {
		group.Add(jen.Return(jen.Id("form").Dot("field" + strcase.ToCamel(field.FieldName))))
	})
	return code, nil
}

func (form *FormUrlencodedRequestStruct) generateSetterFunc(field FormUrlencodedRequestField, options CodeGeneratorOptions) (jen.Code, error) {
	params := jen.Id("value")
	if field.Multiple {
		params = params.Index()
	}
	params, err := field.Type.AddTypeToStatement(params)
	if err != nil {
		return nil, err
	}
	return jen.Func().
		Params(jen.Id("form").Op("*").Id(options.Name)).
		Id("Set" + strcase.ToCamel(field.FieldName)).
		Params(params).
		Params(jen.Op("*").Id(options.Name)).
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Id("form").Dot("field" + strcase.ToCamel(field.FieldName)).Op("=").Id("value"))
			group.Add(jen.Return(jen.Id("form")))
		}), nil
}

func (form *FormUrlencodedRequestStruct) generateSetCall(field FormUrlencodedRequestField) (jen.Code, error) {
	var code *jen.Statement
	fieldName := "field" + strcase.ToCamel(field.FieldName)
	if field.Multiple {
		valueConvertCode, err := field.Type.GenerateConvertCodeToString(jen.Id("value"))
		if err != nil {
			return nil, err
		}
		code = jen.If(jen.Len(jen.Id("form").Dot(fieldName)).Op(">").Lit(0)).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.For(jen.List(jen.Id("_"), jen.Id("value")).Op(":=").Range().Add(jen.Id("form").Dot(fieldName))).BlockFunc(func(group *jen.Group) {
						group.Add(jen.Id("formValues").Dot("Add").Call(jen.Lit(field.Key), valueConvertCode))
					}),
				)
			})
	} else {
		formField := jen.Id("form").Dot(fieldName)
		valueConvertCode, err := field.Type.GenerateConvertCodeToString(formField)
		if err != nil {
			return nil, err
		}
		zeroValue, err := field.Type.ZeroValue()
		if err != nil {
			return nil, err
		}
		condition := formField.Clone()
		if v, ok := zeroValue.(bool); !ok || v {
			condition = condition.Op("!=").Lit(zeroValue)
		}
		code = jen.If(condition).BlockFunc(func(group *jen.Group) {
			group.Add(jen.Id("formValues").Dot("Set").Call(jen.Lit(field.Key), valueConvertCode))
		})

	}
	if !field.Type.IsNumeric() {
		code = code.Else().BlockFunc(func(group *jen.Group) {
			group.Add(jen.Return(
				jen.Nil(),
				jen.Qual("github.com/qiniu/go-sdk/v7/storagev2/errors", "MissingRequiredFieldError").
					ValuesFunc(func(group *jen.Group) {
						group.Add(jen.Id("Name").Op(":").Lit(strcase.ToCamel(field.FieldName)))
					}),
			))
		})
	}
	return code, nil
}

func (form *FormUrlencodedRequestStruct) getServiceBucketField() FormUrlencodedRequestField {
	var serviceBucketField FormUrlencodedRequestField

	for _, field := range form.Fields {
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

func (form *FormUrlencodedRequestStruct) generateServiceBucketField(options CodeGeneratorOptions) jen.Code {
	field := form.getServiceBucketField()
	if field.ServiceBucket.ToServiceBucketType() == ServiceBucketTypeNone {
		return nil
	} else if field.Multiple {
		panic(fmt.Sprintf("multiple service bucket fields: %s", field.FieldName))
	} else if t := field.Type.ToStringLikeType(); t != StringLikeTypeString {
		panic(fmt.Sprintf("service bucket field must be string: %s", t))
	}
	return jen.Func().
		Params(jen.Id("form").Op("*").Id(options.Name)).
		Id("getBucketName").
		Params().
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
					jen.If(jen.Id("putPolicy"), jen.Err()).
						Op(":=").
						Qual("github.com/qiniu/go-sdk/v7/storagev2/uptoken", "NewParser").
						Call(jen.Id("form").Dot("field" + strcase.ToCamel(field.FieldName))).
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
