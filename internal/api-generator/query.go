package main

import (
	"errors"
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
)

type (
	QueryName struct {
		FieldName          string             `yaml:"field_name,omitempty"`
		FieldCamelCaseName string             `yaml:"field_camel_case_name,omitempty"`
		FieldSnakeCaseName string             `yaml:"field_snake_case_name,omitempty"`
		QueryName          string             `yaml:"query_name,omitempty"`
		Documentation      string             `yaml:"documentation,omitempty"`
		Multiple           bool               `yaml:"multiple,omitempty"`
		QueryType          *StringLikeType    `yaml:"query_type,omitempty"`
		ServiceBucket      *ServiceBucketType `yaml:"service_bucket,omitempty"`
		ServiceObject      *ServiceObjectType `yaml:"service_object,omitempty"`
		Optional           *OptionalType      `yaml:"optional,omitempty"`
	}
	QueryNames []QueryName
)

func (name *QueryName) camelCaseName() string {
	if name.FieldCamelCaseName != "" {
		return name.FieldCamelCaseName
	}
	return strcase.ToCamel(name.FieldName)
}

func (names QueryNames) addFields(group *jen.Group) error {
	for _, queryName := range names {
		code := jen.Id(queryName.camelCaseName())
		if queryName.Multiple {
			code = code.Index()
		}
		code, err := queryName.QueryType.AddTypeToStatement(code, false)
		if err != nil {
			return err
		}
		if queryName.Documentation != "" {
			code = code.Comment(queryName.Documentation)
		}
		group.Add(code)
	}
	return nil
}

func (names QueryNames) addGetBucketNameFunc(group *jen.Group, structName string) (bool, error) {
	field := names.getServiceBucketField()
	if field == nil || field.ServiceBucket.ToServiceBucketType() == ServiceBucketTypeNone {
		return false, nil
	} else if field.Multiple {
		panic(fmt.Sprintf("multiple service bucket fields: %s", field.FieldName))
	} else if field.QueryType.ToStringLikeType() != StringLikeTypeString {
		panic(fmt.Sprintf("service bucket field must be string: %s", field.FieldName))
	}
	group.Add(jen.Func().
		Params(jen.Id("query").Op("*").Id(structName)).
		Id("getBucketName").
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(jen.String(), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			fieldName := field.camelCaseName()
			switch field.ServiceBucket.ToServiceBucketType() {
			case ServiceBucketTypePlainText:
				group.Add(jen.Return(jen.Id("query").Dot(fieldName), jen.Nil()))
			case ServiceBucketTypeEntry:
				group.Add(
					jen.Return(
						jen.Qual("strings", "SplitN").
							Call(jen.Id("query").Dot(fieldName), jen.Lit(":"), jen.Lit(2)).
							Index(jen.Lit(0)),
						jen.Nil(),
					),
				)
			case ServiceBucketTypeUploadToken:
				group.Add(
					jen.If(jen.Id("putPolicy"), jen.Err()).
						Op(":=").
						Qual(PackageNameUpToken, "NewParser").
						Call(jen.Id("query").Dot(fieldName)).
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

func (names QueryNames) addGetObjectNameFunc(group *jen.Group, structName string) (bool, error) {
	field := names.getServiceObjectField()
	if field == nil || field.ServiceObject.ToServiceObjectType() == ServiceObjectTypeNone {
		return false, nil
	} else if field.Multiple {
		panic(fmt.Sprintf("multiple service object fields: %s", field.FieldName))
	} else if field.QueryType.ToStringLikeType() != StringLikeTypeString {
		panic(fmt.Sprintf("service object field must be string: %s", field.FieldName))
	}
	group.Add(jen.Func().
		Params(jen.Id("query").Op("*").Id(structName)).
		Id("getObjectName").
		Params().
		Params(jen.String()).
		BlockFunc(func(group *jen.Group) {
			fieldName := field.camelCaseName()
			switch field.ServiceObject.ToServiceObjectType() {
			case ServiceObjectTypePlainText:
				if field.Optional.ToOptionalType() == OptionalTypeNullable {
					group.Add(jen.Var().Id("objectName").String())
					group.Add(jen.If(jen.Id("query").Dot(fieldName).Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
						group.Add(jen.Id("objectName").Op("=").Op("*").Id("query").Dot(fieldName))
					}))
					group.Add(jen.Return(jen.Id("objectName")))
				} else {
					group.Return(jen.Id("query").Dot(fieldName))
				}
			case ServiceObjectTypeEntry:
				group.Add(
					jen.Id("parts").
						Op(":=").
						Qual("strings", "SplitN").
						Call(jen.Id("query").Dot(fieldName), jen.Lit(":"), jen.Lit(2)))
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

func (names QueryNames) addBuildFunc(group *jen.Group, structName string) error {
	var err error
	group.Add(
		jen.Func().
			Params(jen.Id("query").Op("*").Id(structName)).
			Id("buildQuery").
			Params().
			Params(jen.Qual("net/url", "Values"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.Id("allQuery").Op(":=").Make(jen.Qual("net/url", "Values")),
				)
				for _, queryName := range names {
					if e := names.generateSetCall(group, queryName, "allQuery"); e != nil {
						err = e
						return
					}
				}
				group.Add(jen.Return(jen.Id("allQuery"), jen.Nil()))
			}),
	)
	return err
}

func (names QueryNames) generateSetCall(group *jen.Group, queryName QueryName, queryVarName string) error {
	var (
		valueConvertCode                    *jen.Statement
		err                                 error
		appendMissingRequiredFieldErrorFunc = func(fieldName string) func(group *jen.Group) {
			return func(group *jen.Group) {
				group.Return(
					jen.Nil(),
					jen.Qual(PackageNameErrors, "MissingRequiredFieldError").
						ValuesFunc(func(group *jen.Group) {
							group.Add(jen.Id("Name").Op(":").Lit(fieldName))
						}),
				)
			}
		}
	)
	fieldName := queryName.camelCaseName()
	field := jen.Id("query").Dot(fieldName)
	if queryName.Multiple {
		valueConvertCode, err = queryName.QueryType.GenerateConvertCodeToString(jen.Id("value"))
		if err != nil {
			return err
		}
		code := jen.If(jen.Len(jen.Id("query").Dot(fieldName)).Op(">").Lit(0)).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.For(jen.List(jen.Id("_"), jen.Id("value")).Op(":=").Range().Add(jen.Id("query").Dot(fieldName))).BlockFunc(func(group *jen.Group) {
						group.Add(jen.Id(queryVarName).Dot("Add").Call(jen.Lit(queryName.QueryName), valueConvertCode))
					}),
				)
			})
		if queryName.Optional.ToOptionalType() == OptionalTypeRequired {
			code = code.Else().BlockFunc(appendMissingRequiredFieldErrorFunc(fieldName))
		}
		group.Add(code)
	} else {
		if queryName.Optional.ToOptionalType() == OptionalTypeNullable {
			valueConvertCode, err = queryName.QueryType.GenerateConvertCodeToString(jen.Op("*").Add(field))
		} else {
			valueConvertCode, err = queryName.QueryType.GenerateConvertCodeToString(field)
		}
		if err != nil {
			return err
		}
		zeroValue, err := queryName.QueryType.ZeroValue()
		if err != nil {
			return err
		}

		condition := field.Clone()
		if queryName.Optional.ToOptionalType() == OptionalTypeNullable {
			condition = condition.Op("!=").Nil()
		} else if v, ok := zeroValue.(bool); !ok || v {
			condition = condition.Op("!=").Lit(zeroValue)
		}
		setQueryFunc := func(queryName string, value jen.Code) func(group *jen.Group) {
			return func(group *jen.Group) {
				group.Id(queryVarName).Dot("Set").Call(jen.Lit(queryName), value)
			}
		}
		switch queryName.Optional.ToOptionalType() {
		case OptionalTypeRequired:
			group.Add(jen.If(condition).
				BlockFunc(setQueryFunc(queryName.QueryName, valueConvertCode)).
				Else().
				BlockFunc(appendMissingRequiredFieldErrorFunc(fieldName)))
		case OptionalTypeOmitEmpty, OptionalTypeNullable:
			group.Add(jen.If(condition).
				BlockFunc(setQueryFunc(queryName.QueryName, valueConvertCode)))
		case OptionalTypeKeepEmpty:
			setQueryFunc(queryName.QueryName, valueConvertCode)(group)
		default:
			return errors.New("unknown OptionalType")
		}
	}
	return nil
}

func (names QueryNames) getServiceBucketField() *QueryName {
	var serviceBucketField *QueryName

	for i := range names {
		if names[i].ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
			if serviceBucketField == nil {
				serviceBucketField = &names[i]
			} else {
				panic(fmt.Sprintf("multiple service bucket fields: %s & %s", names[i].FieldName, serviceBucketField.FieldName))
			}
		}
	}
	return serviceBucketField
}

func (names QueryNames) getServiceObjectField() *QueryName {
	var serviceObjectField *QueryName

	for i := range names {
		if names[i].ServiceObject.ToServiceObjectType() != ServiceObjectTypeNone {
			if serviceObjectField == nil {
				serviceObjectField = &names[i]
			} else {
				panic(fmt.Sprintf("multiple service object fields: %s & %s", names[i].FieldName, serviceObjectField.FieldName))
			}
		}
	}
	return serviceObjectField
}
