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
		FieldName          string             `yaml:"field_name,omitempty"`
		FieldCamelCaseName string             `yaml:"field_camel_case_name,omitempty"`
		FieldSnakeCaseName string             `yaml:"field_snake_case_name,omitempty"`
		Key                string             `yaml:"key,omitempty"`
		Documentation      string             `yaml:"documentation,omitempty"`
		Type               *StringLikeType    `yaml:"type,omitempty"`
		Multiple           bool               `yaml:"multiple,omitempty"`
		Optional           *OptionalType      `yaml:"optional,omitempty"`
		ServiceBucket      *ServiceBucketType `yaml:"service_bucket,omitempty"`
		ServiceObject      *ServiceObjectType `yaml:"service_object,omitempty"`
	}
)

func (field *FormUrlencodedRequestField) camelCaseName() string {
	if field.FieldCamelCaseName != "" {
		return field.FieldCamelCaseName
	}
	return strcase.ToCamel(field.FieldName)
}

func (form *FormUrlencodedRequestStruct) addFields(group *jen.Group) error {
	for _, field := range form.Fields {
		if err := form.generateField(group, field); err != nil {
			return err
		}
	}
	return nil
}

func (form *FormUrlencodedRequestStruct) addGetBucketNameFunc(group *jen.Group, structName string) (bool, error) {
	field := form.getServiceBucketField()
	if field == nil || field.ServiceBucket.ToServiceBucketType() == ServiceBucketTypeNone {
		return false, nil
	} else if field.Multiple {
		panic(fmt.Sprintf("multiple service bucket fields: %s", field.FieldName))
	} else if t := field.Type.ToStringLikeType(); t != StringLikeTypeString {
		panic(fmt.Sprintf("service bucket field must be string: %s", t))
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
					jen.If(jen.Id("putPolicy"), jen.Err()).
						Op(":=").
						Qual(PackageNameUpToken, "NewParser").
						Call(jen.Id("form").Dot(fieldName)).
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

func (form *FormUrlencodedRequestStruct) addGetObjectNameFunc(group *jen.Group, structName string) (bool, error) {
	field := form.getServiceObjectField()
	if field == nil || field.ServiceObject.ToServiceObjectType() == ServiceObjectTypeNone {
		return false, nil
	} else if field.Multiple {
		panic(fmt.Sprintf("multiple service object fields: %s", field.FieldName))
	} else if t := field.Type.ToStringLikeType(); t != StringLikeTypeString {
		panic(fmt.Sprintf("service object field must be string: %s", t))
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

func (form *FormUrlencodedRequestStruct) addBuildFunc(group *jen.Group, structName string) error {
	var finalErr error = nil
	group.Add(
		jen.Func().
			Params(jen.Id("form").Op("*").Id(structName)).
			Id("build").
			Params().
			Params(jen.Qual("net/url", "Values"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.Id("formValues").Op(":=").Make(jen.Qual("net/url", "Values")))
				for _, field := range form.Fields {
					if err := form.addSetCall(group, field, "form", "formValues"); err != nil {
						finalErr = err
						return
					}
				}
				group.Add(jen.Return(jen.Id("formValues"), jen.Nil()))
			}),
	)
	return finalErr
}

func (form *FormUrlencodedRequestStruct) getServiceBucketField() *FormUrlencodedRequestField {
	var serviceBucketField *FormUrlencodedRequestField

	for i := range form.Fields {
		if form.Fields[i].ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
			if serviceBucketField == nil {
				serviceBucketField = &form.Fields[i]
			} else {
				panic(fmt.Sprintf("multiple service bucket fields: %s & %s", form.Fields[i].FieldName, serviceBucketField.FieldName))
			}
		}
	}
	return serviceBucketField
}

func (form *FormUrlencodedRequestStruct) getServiceObjectField() *FormUrlencodedRequestField {
	var serviceObjectField *FormUrlencodedRequestField

	for i := range form.Fields {
		if form.Fields[i].ServiceObject.ToServiceObjectType() != ServiceBucketTypeNone {
			if serviceObjectField == nil {
				serviceObjectField = &form.Fields[i]
			} else {
				panic(fmt.Sprintf("multiple service object fields: %s & %s", form.Fields[i].FieldName, serviceObjectField.FieldName))
			}
		}
	}
	return serviceObjectField
}

func (form *FormUrlencodedRequestStruct) generateField(group *jen.Group, field FormUrlencodedRequestField) error {
	code := jen.Id(field.camelCaseName())
	if field.Multiple {
		code = code.Index()
	}
	code, err := field.Type.AddTypeToStatement(code, field.Optional.ToOptionalType() == OptionalTypeNullable)
	if err != nil {
		return err
	}
	if field.Documentation != "" {
		code = code.Comment(field.Documentation)
	}
	group.Add(code)
	return nil
}

func (form *FormUrlencodedRequestStruct) addSetCall(group *jen.Group, field FormUrlencodedRequestField, formVarName, formValuesVarName string) error {
	var (
		code, valueConvertCode *jen.Statement
		err                    error
	)
	fieldName := field.camelCaseName()
	if field.Multiple {
		valueConvertCode, err = field.Type.GenerateConvertCodeToString(jen.Id("value"))
		if err != nil {
			return err
		}
		code = jen.If(jen.Len(jen.Id(formVarName).Dot(fieldName)).Op(">").Lit(0)).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.For(jen.List(jen.Id("_"), jen.Id("value")).Op(":=").Range().Add(jen.Id(formVarName).Dot(fieldName))).BlockFunc(func(group *jen.Group) {
						group.Add(jen.Id(formValuesVarName).Dot("Add").Call(jen.Lit(field.Key), valueConvertCode))
					}),
				)
			})
	} else {
		formField := jen.Id(formVarName).Dot(fieldName)
		if field.Optional.ToOptionalType() == OptionalTypeNullable {
			valueConvertCode, err = field.Type.GenerateConvertCodeToString(jen.Op("*").Add(formField))
		} else {
			valueConvertCode, err = field.Type.GenerateConvertCodeToString(formField)
		}
		if err != nil {
			return err
		}
		zeroValue, err := field.Type.ZeroValue()
		if err != nil {
			return err
		}
		condition := formField.Clone()
		if field.Optional.ToOptionalType() == OptionalTypeNullable {
			condition = condition.Op("!=").Nil()
		} else if v, ok := zeroValue.(bool); !ok || v {
			condition = condition.Op("!=").Lit(zeroValue)
		}
		switch field.Optional.ToOptionalType() {
		case OptionalTypeOmitEmpty, OptionalTypeRequired, OptionalTypeNullable:
			code = jen.If(condition).BlockFunc(func(group *jen.Group) {
				group.Add(jen.Id(formValuesVarName).Dot("Set").Call(jen.Lit(field.Key), valueConvertCode))
			})
		case OptionalTypeKeepEmpty:
			code = jen.Id(formValuesVarName).Dot("Set").Call(jen.Lit(field.Key), valueConvertCode)
		}
	}
	if field.Optional.ToOptionalType() == OptionalTypeRequired {
		code = code.Else().BlockFunc(func(group *jen.Group) {
			group.Add(jen.Return(
				jen.Nil(),
				jen.Qual(PackageNameErrors, "MissingRequiredFieldError").
					ValuesFunc(func(group *jen.Group) {
						group.Add(jen.Id("Name").Op(":").Lit(field.camelCaseName()))
					}),
			))
		})
	}
	group.Add(code)
	return nil
}
