package main

import (
	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
)

type (
	HeaderName struct {
		FieldName     string        `yaml:"field_name,omitempty"`
		HeaderName    string        `yaml:"header_name,omitempty"`
		Documentation string        `yaml:"documentation,omitempty"`
		Optional      *OptionalType `yaml:"optional,omitempty"`
	}
	HeaderNames []HeaderName
)

func (names HeaderNames) Generate(group *jen.Group, options CodeGeneratorOptions) error {
	if len(names) == 0 {
		return nil
	}

	options.Name = strcase.ToCamel(options.Name)
	if options.Documentation != "" {
		group.Add(jen.Comment(options.Documentation))
	}
	group.Add(
		jen.Type().Id(options.Name).StructFunc(func(group *jen.Group) {
			for _, headerName := range names {
				code := jen.Id("field" + strcase.ToCamel(headerName.FieldName)).String()
				if headerName.Documentation != "" {
					code = code.Comment(headerName.Documentation)
				}
				group.Add(code)
			}
		}),
	)
	for _, name := range names {
		names.generateGetterFunc(group, name, options)
		names.generateSetterFunc(group, name, options)
	}
	group.Add(
		jen.Func().
			Params(jen.Id("headers").Op("*").Id(options.Name)).
			Id("build").
			Params().
			Params(jen.Qual("net/http", "Header"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.Id("allHeaders").Op(":=").Make(jen.Qual("net/http", "Header")),
				)
				for _, headerName := range names {
					fieldName := "field" + strcase.ToCamel(headerName.FieldName)
					cond := jen.Id("headers").Dot(fieldName).Op("!=").Lit("")
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
					case OptionalTypeOmitEmpty:
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

func (names HeaderNames) GenerateAliasesFor(group *jen.Group, structName, fieldName string) error {
	for _, name := range names {
		names.generateAliasGetterFunc(group, name, structName, fieldName)
		names.generateAliasSetterFunc(group, name, structName, fieldName)
	}
	return nil
}

func (names HeaderNames) generateAliasGetterFunc(group *jen.Group, name HeaderName, structName, fieldName string) {
	if name.Documentation != "" {
		group.Add(jen.Comment(name.Documentation))
	}
	group.Add(jen.Func().
		Params(jen.Id("request").Op("*").Id(structName)).
		Id(makeGetterMethodName(name.FieldName)).
		Params().
		String().
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Return(jen.Id("request").Dot(fieldName).Dot(makeGetterMethodName(name.FieldName)).Call()))
		}))
}

func (names HeaderNames) generateAliasSetterFunc(group *jen.Group, name HeaderName, structName, fieldName string) {
	if name.Documentation != "" {
		group.Add(jen.Comment(name.Documentation))
	}
	group.Add(jen.Func().
		Params(jen.Id("request").Op("*").Id(structName)).
		Id(makeSetterMethodName(name.FieldName)).
		Params(jen.Id("value").String()).
		Params(jen.Op("*").Id(structName)).
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Id("request").Dot(fieldName).Dot(makeSetterMethodName(name.FieldName)).Call(jen.Id("value")))
			group.Add(jen.Return(jen.Id("request")))
		}))
}

func (names HeaderNames) generateGetterFunc(group *jen.Group, name HeaderName, options CodeGeneratorOptions) {
	if name.Documentation != "" {
		group.Add(jen.Comment(name.Documentation))
	}
	group.Add(jen.Func().
		Params(jen.Id("header").Op("*").Id(options.Name)).
		Id(makeGetterMethodName(name.FieldName)).
		Params().
		String().
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Return(jen.Id("header").Dot("field" + strcase.ToCamel(name.FieldName))))
		}))
}

func (names HeaderNames) generateSetterFunc(group *jen.Group, name HeaderName, options CodeGeneratorOptions) {
	if name.Documentation != "" {
		group.Add(jen.Comment(name.Documentation))
	}
	group.Add(jen.Func().
		Params(jen.Id("header").Op("*").Id(options.Name)).
		Id(makeSetterMethodName(name.FieldName)).
		Params(jen.Id("value").String()).
		Params(jen.Op("*").Id(options.Name)).
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Id("header").Dot("field" + strcase.ToCamel(name.FieldName)).Op("=").Id("value"))
			group.Add(jen.Return(jen.Id("header")))
		}))
}
