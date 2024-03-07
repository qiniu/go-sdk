package main

import (
	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
)

type (
	HeaderName struct {
		FieldName          string        `yaml:"field_name,omitempty"`
		FieldCamelCaseName string        `yaml:"field_camel_case_name,omitempty"`
		FieldSnakeCaseName string        `yaml:"field_snake_case_name,omitempty"`
		HeaderName         string        `yaml:"header_name,omitempty"`
		Documentation      string        `yaml:"documentation,omitempty"`
		Optional           *OptionalType `yaml:"optional,omitempty"`
	}
	HeaderNames []HeaderName
)

func (name *HeaderName) camelCaseName() string {
	if name.FieldCamelCaseName != "" {
		return name.FieldCamelCaseName
	}
	return strcase.ToCamel(name.FieldName)
}

func (names HeaderNames) addFields(group *jen.Group) error {
	for _, headerName := range names {
		code := jen.Id(headerName.camelCaseName())
		if headerName.Optional.ToOptionalType() == OptionalTypeNullable {
			code = code.Op("*")
		}
		code = code.String()
		if headerName.Documentation != "" {
			code = code.Comment(headerName.Documentation)
		}
		group.Add(code)
	}
	return nil
}

func (names HeaderNames) addGetBucketNameFunc(group *jen.Group, structName string) (bool, error) {
	return false, nil
}

func (names HeaderNames) addGetObjectNameFunc(group *jen.Group, structName string) (bool, error) {
	return false, nil
}

func (names HeaderNames) addBuildFunc(group *jen.Group, structName string) error {
	group.Add(
		jen.Func().
			Params(jen.Id("headers").Op("*").Id(structName)).
			Id("buildHeaders").
			Params().
			Params(jen.Qual("net/http", "Header"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.Id("allHeaders").Op(":=").Make(jen.Qual("net/http", "Header")),
				)
				for _, headerName := range names {
					fieldName := headerName.camelCaseName()
					cond := jen.Id("headers").Dot(fieldName)
					if headerName.Optional.ToOptionalType() == OptionalTypeNullable {
						cond = cond.Op("!=").Nil()
					} else {
						cond = cond.Op("!=").Lit("")
					}
					setHeaderFunc := func(headerName, fieldName string) func(*jen.Group) {
						return func(group *jen.Group) {
							group.Add(jen.Id("allHeaders").Dot("Set").Call(jen.Lit(headerName), jen.Id("headers").Dot(fieldName)))
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
					switch headerName.Optional.ToOptionalType() {
					case OptionalTypeRequired:
						group.Add(
							jen.If(cond).
								BlockFunc(setHeaderFunc(headerName.HeaderName, fieldName)).
								Else().
								BlockFunc(appendMissingRequiredFieldErrorFunc(fieldName)),
						)
					case OptionalTypeOmitEmpty, OptionalTypeNullable:
						group.Add(
							jen.If(cond).
								BlockFunc(setHeaderFunc(headerName.HeaderName, fieldName)),
						)
					case OptionalTypeKeepEmpty:
						setHeaderFunc(headerName.HeaderName, fieldName)(group)
					}
				}
				group.Add(jen.Return(jen.Id("allHeaders"), jen.Nil()))
			}),
	)
	return nil
}
