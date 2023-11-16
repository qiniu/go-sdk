package main

import (
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
)

type (
	QueryName struct {
		FieldName     string             `yaml:"field_name,omitempty"`
		QueryName     string             `yaml:"query_name,omitempty"`
		Documentation string             `yaml:"documentation,omitempty"`
		QueryType     *StringLikeType    `yaml:"query_type,omitempty"`
		ServiceBucket *ServiceBucketType `yaml:"service_bucket,omitempty"`
		Optional      bool               `yaml:"optional,omitempty"`
	}
	QueryNames []QueryName
)

func (names QueryNames) Generate(group *jen.Group, options CodeGeneratorOptions) error {
	var err error

	if len(names) == 0 {
		return nil
	}

	options.Name = strcase.ToCamel(options.Name)
	if options.Documentation != "" {
		group.Add(jen.Comment(options.Documentation))
	}
	group.Add(
		jen.Type().Id(options.Name).StructFunc(func(group *jen.Group) {
			for _, queryName := range names {
				if code, e := names.generateField(queryName, options); e != nil {
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
	for _, name := range names {
		if code, err := names.generateGetterFunc(name, options); err != nil {
			return err
		} else {
			group.Add(code)
		}
		if code, err := names.generateSetterFunc(name, options); err != nil {
			return err
		} else {
			group.Add(code)
		}
	}
	if code := names.generateServiceBucketField(options); code != nil {
		group.Add(code)
	}

	group.Add(
		jen.Func().
			Params(jen.Id("query").Op("*").Id(options.Name)).
			Id("build").
			Params().
			Params(jen.Qual("net/url", "Values"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.Id("allQuery").Op(":=").Make(jen.Qual("net/url", "Values")),
				)
				for _, queryName := range names {
					if code, e := names.generateSetCall(queryName, options); e != nil {
						err = e
						return
					} else {
						group.Add(code)
					}
				}
				group.Add(jen.Return(jen.Id("allQuery"), jen.Nil()))
			}),
	)

	return err
}

func (names QueryNames) generateField(queryName QueryName, options CodeGeneratorOptions) (jen.Code, error) {
	code, err := queryName.QueryType.AddTypeToStatement(jen.Id("field" + strcase.ToCamel(queryName.FieldName)))
	if err != nil {
		return nil, err
	}
	code = code.Comment(queryName.Documentation)
	return code, nil
}

func (names QueryNames) generateGetterFunc(queryName QueryName, options CodeGeneratorOptions) (jen.Code, error) {
	code := jen.Func().
		Params(jen.Id("query").Op("*").Id(options.Name)).
		Id("Get" + strcase.ToCamel(queryName.FieldName)).
		Params()
	code, err := queryName.QueryType.AddTypeToStatement(code)
	if err != nil {
		return nil, err
	}
	code = code.BlockFunc(func(group *jen.Group) {
		group.Add(jen.Return(jen.Id("query").Dot("field" + strcase.ToCamel(queryName.FieldName))))
	})
	return code, nil
}

func (names QueryNames) generateSetterFunc(queryName QueryName, options CodeGeneratorOptions) (jen.Code, error) {
	params, err := queryName.QueryType.AddTypeToStatement(jen.Id("value"))
	if err != nil {
		return nil, err
	}
	return jen.Func().
		Params(jen.Id("query").Op("*").Id(options.Name)).
		Id("Set" + strcase.ToCamel(queryName.FieldName)).
		Params(params).
		Params(jen.Op("*").Id(options.Name)).
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Id("query").Dot("field" + strcase.ToCamel(queryName.FieldName)).Op("=").Id("value"))
			group.Add(jen.Return(jen.Id("query")))
		}), nil
}

func (names QueryNames) generateSetCall(queryName QueryName, options CodeGeneratorOptions) (jen.Code, error) {
	fieldName := strcase.ToCamel(queryName.FieldName)
	field := jen.Id("query").Dot("field" + fieldName)
	valueConvertCode, err := queryName.QueryType.GenerateConvertCodeToString(field)
	if err != nil {
		return nil, err
	}
	zeroValue, err := queryName.QueryType.ZeroValue()
	if err != nil {
		return nil, err
	}

	condition := field.Clone()
	if v, ok := zeroValue.(bool); !ok || v {
		condition = condition.Op("!=").Lit(zeroValue)
	}
	code := jen.If(condition).BlockFunc(func(group *jen.Group) {
		group.Add(jen.Id("allQuery").Dot("Set").Call(jen.Lit(queryName.QueryName), valueConvertCode))
	})
	if !queryName.Optional && !queryName.QueryType.IsNumeric() {
		code = code.Else().BlockFunc(func(group *jen.Group) {
			group.Add(jen.Return(
				jen.Nil(),
				jen.Qual("github.com/qiniu/go-sdk/v7/storagev2/errors", "MissingRequiredFieldError").
					ValuesFunc(func(group *jen.Group) {
						group.Add(jen.Id("Name").Op(":").Lit(fieldName))
					}),
			))
		})
	}
	return code, nil
}

func (names QueryNames) getServiceBucketField() QueryName {
	var serviceBucketField QueryName

	for _, field := range names {
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

func (names QueryNames) generateServiceBucketField(options CodeGeneratorOptions) jen.Code {
	field := names.getServiceBucketField()
	if field.ServiceBucket.ToServiceBucketType() == ServiceBucketTypeNone {
		return nil
	} else if field.QueryType.ToStringLikeType() != StringLikeTypeString {
		panic(fmt.Sprintf("service bucket field must be string: %s", field.FieldName))
	}
	return jen.Func().
		Params(jen.Id("query").Op("*").Id(options.Name)).
		Id("getBucketName").
		Params().
		Params(jen.String(), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			switch field.ServiceBucket.ToServiceBucketType() {
			case ServiceBucketTypePlainText:
				group.Add(jen.Return(jen.Id("query").Dot("field"+strcase.ToCamel(field.FieldName)), jen.Nil()))
			case ServiceBucketTypeEntry:
				group.Add(
					jen.Return(
						jen.Qual("strings", "SplitN").
							Call(jen.Id("query").Dot("field"+strcase.ToCamel(field.FieldName)), jen.Lit(":"), jen.Lit(2)).
							Index(jen.Lit(0)),
						jen.Nil(),
					),
				)
			case ServiceBucketTypeUploadToken:
				group.Add(
					jen.If(jen.Id("putPolicy"), jen.Err()).
						Op(":=").
						Qual("github.com/qiniu/go-sdk/v7/storagev2/uptoken", "NewParser").
						Call(jen.Id("query").Dot("field" + strcase.ToCamel(field.FieldName))).
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
