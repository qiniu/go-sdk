package main

import (
	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
)

type (
	HeaderName struct {
		FieldName     string `yaml:"field_name,omitempty"`
		HeaderName    string `yaml:"header_name,omitempty"`
		Documentation string `yaml:"documentation,omitempty"`
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
		group.Add(names.generateGetterFunc(name, options))
		group.Add(names.generateSetterFunc(name, options))
	}
	group.Add(
		jen.Func().
			Params(jen.Id("headers").Op("*").Id(options.Name)).
			Id("build").
			Params().
			Params(jen.Qual("net/http", "Header")).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.Id("allHeaders").Op(":=").Make(jen.Qual("net/http", "Header")),
				)
				for _, headerName := range names {
					fieldName := "field" + strcase.ToCamel(headerName.FieldName)
					group.Add(
						jen.If(jen.Id("headers").Dot(fieldName).Op("!=").Lit("")).BlockFunc(func(group *jen.Group) {
							group.Add(jen.Id("allHeaders").Dot("Set").Call(jen.Lit(headerName.HeaderName), jen.Id("headers").Dot(fieldName)))
						}),
					)
				}
				group.Add(jen.Return(jen.Id("allHeaders")))
			}),
	)

	return nil
}

func (names HeaderNames) generateGetterFunc(name HeaderName, options CodeGeneratorOptions) jen.Code {
	return jen.Func().
		Params(jen.Id("header").Op("*").Id(options.Name)).
		Id("Get" + strcase.ToCamel(name.FieldName)).
		Params().
		String().
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Return(jen.Id("header").Dot("field" + strcase.ToCamel(name.FieldName))))
		})
}

func (names HeaderNames) generateSetterFunc(name HeaderName, options CodeGeneratorOptions) jen.Code {
	return jen.Func().
		Params(jen.Id("header").Op("*").Id(options.Name)).
		Id("Set" + strcase.ToCamel(name.FieldName)).
		Params(jen.Id("value").String()).
		Params(jen.Op("*").Id(options.Name)).
		BlockFunc(func(group *jen.Group) {
			group.Add(jen.Id("header").Dot("field" + strcase.ToCamel(name.FieldName)).Op("=").Id("value"))
			group.Add(jen.Return(jen.Id("header")))
		})
}
