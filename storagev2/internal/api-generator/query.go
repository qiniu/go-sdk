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
		QueryType          *StringLikeType    `yaml:"query_type,omitempty"`
		ServiceBucket      *ServiceBucketType `yaml:"service_bucket,omitempty"`
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
		code, err := queryName.QueryType.AddTypeToStatement(jen.Id(queryName.camelCaseName()), false)
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
					if e := names.generateSetCall(group, queryName); e != nil {
						err = e
						return
					}
				}
				group.Add(jen.Return(jen.Id("allQuery"), jen.Nil()))
			}),
	)
	return err
}

func (names QueryNames) generateSetCall(group *jen.Group, queryName QueryName) error {
	var (
		valueConvertCode *jen.Statement
		err              error
	)
	fieldName := queryName.camelCaseName()
	field := jen.Id("query").Dot(fieldName)
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
			group.Id("allQuery").Dot("Set").Call(jen.Lit(queryName), value)
		}
	}
	appendMissingRequiredFieldErrorFunc := func(fieldName string) func(group *jen.Group) {
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
