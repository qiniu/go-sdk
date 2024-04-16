package main

import (
	"errors"
	"fmt"

	"github.com/dave/jennifer/jen"
	"gopkg.in/yaml.v3"
)

type (
	ApiRequestDescription struct {
		PathParams           *PathParams    `yaml:"path_params,omitempty"`
		HeaderNames          HeaderNames    `yaml:"header_names,omitempty"`
		QueryNames           QueryNames     `yaml:"query_names,omitempty"`
		Body                 *RequestBody   `yaml:"body,omitempty"`
		Authorization        *Authorization `yaml:"authorization,omitempty"`
		Idempotent           *Idempotent    `yaml:"idempotent,omitempty"`
		responseTypeRequired bool
	}

	RequestBody struct {
		Json              *JsonType
		FormUrlencoded    *FormUrlencodedRequestStruct
		MultipartFormData *MultipartFormFields
		BinaryData        bool
	}
)

func (request *ApiRequestDescription) generate(group *jen.Group, opts CodeGeneratorOptions) (err error) {
	if opts.Documentation != "" {
		group.Add(jen.Comment(opts.Documentation))
	}
	group.Add(jen.Type().
		Id(opts.camelCaseName()).
		StructFunc(func(group *jen.Group) {
			err = request.addFields(group)
		}))
	if err != nil {
		return
	}

	if body := request.Body; body != nil {
		if bodyJson := body.Json; bodyJson != nil {
			err = bodyJson.generate(group, opts)
		}
	}

	return
}

func (request *ApiRequestDescription) addFields(group *jen.Group) (err error) {
	if pp := request.PathParams; pp != nil {
		if err = pp.addFields(group); err != nil {
			return
		}
	}
	if names := request.QueryNames; names != nil {
		if err = names.addFields(group); err != nil {
			return
		}
	}
	if names := request.HeaderNames; names != nil {
		if err = names.addFields(group); err != nil {
			return
		}
	}
	if authorization := request.Authorization; authorization != nil {
		switch authorization.ToAuthorization() {
		case AuthorizationQbox, AuthorizationQiniu:
			group.Add(jen.Id("Credentials").Qual(PackageNameCredentials, "CredentialsProvider").Comment("鉴权参数，用于生成鉴权凭证，如果为空，则使用 HTTPClientOptions 中的 CredentialsProvider"))
		case AuthorizationUpToken:
			group.Add(jen.Id("UpToken").Qual(PackageNameUpToken, "Provider").Comment("上传凭证，如果为空，则使用 HTTPClientOptions 中的 UpToken"))
		}
	}
	if body := request.Body; body != nil {
		if bodyJson := body.Json; bodyJson != nil {
			if jsonStruct := bodyJson.Struct; jsonStruct != nil {
				if err = jsonStruct.addFields(group, false); err != nil {
					return
				}
			} else if jsonArray := bodyJson.Array; jsonArray != nil {
				if err = jsonArray.addFields(group); err != nil {
					return
				}
			} else {
				return errors.New("request body should be struct or array")
			}
		} else if formUrlencoded := body.FormUrlencoded; formUrlencoded != nil {
			if err = formUrlencoded.addFields(group); err != nil {
				return
			}
		} else if multipartFormData := body.MultipartFormData; multipartFormData != nil {
			if err = multipartFormData.addFields(group); err != nil {
				return
			}
		} else if body.BinaryData {
			group.Add(jen.Id("Body").Qual(PackageNameInternalIo, "ReadSeekCloser").Comment("请求体"))
		}
	}
	if request.responseTypeRequired {
		group.Add(jen.Id("ResponseBody").Interface().Comment("响应体，如果为空，则 Response.Body 的类型由 encoding/json 库决定"))
	}
	return
}

func (request *ApiRequestDescription) generateGetAccessKeyFunc(group *jen.Group, structName string) (err error) {
	group.Add(
		jen.Func().
			Params(jen.Id("request").Op("*").Id(structName)).
			Id("getAccessKey").
			Params(jen.Id("ctx").Qual("context", "Context")).
			Params(jen.Id("string"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				if authorization := request.Authorization; authorization != nil {
					switch authorization.ToAuthorization() {
					case AuthorizationQbox, AuthorizationQiniu:
						group.Add(
							jen.If(jen.Id("request").Dot("Credentials").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
								group.Add(
									jen.If(
										jen.List(jen.Id("credentials"), jen.Err()).
											Op(":=").
											Id("request").
											Dot("Credentials").
											Dot("Get").
											Call(jen.Id("ctx")),
										jen.Err().Op("!=").Nil(),
									).BlockFunc(func(group *jen.Group) {
										group.Return(jen.Lit(""), jen.Err())
									}).Else().BlockFunc(func(group *jen.Group) {
										group.Return(jen.Id("credentials").Dot("AccessKey"), jen.Nil())
									}),
								)
							}),
						)
					case AuthorizationUpToken:
						group.Add(
							jen.If(jen.Id("request").Dot("UpToken").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
								group.Return(jen.Id("request").Dot("UpToken").Dot("GetAccessKey").Call(jen.Id("ctx")))
							}),
						)
					}
				}
				if body := request.Body; body != nil {
					if multipartForm := body.MultipartFormData; multipartForm != nil {
						if field := multipartForm.getServiceBucketField(); field != nil && field.ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
							fieldName := field.camelCaseName()
							group.Add(
								jen.If(jen.Id("request").Dot(fieldName).Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
									group.If(
										jen.List(jen.Id("accessKey"), jen.Err()).
											Op(":=").
											Id("request").
											Dot(fieldName).
											Dot("GetAccessKey").
											Call(jen.Id("ctx")),
										jen.Err().Op("!=").Nil(),
									).BlockFunc(func(group *jen.Group) {
										group.Add(jen.Return(jen.Lit(""), jen.Err()))
									}).Else().BlockFunc(func(group *jen.Group) {
										group.Add(jen.Return(jen.Id("accessKey"), jen.Nil()))
									})
								}),
							)
						}
					}
				}
				group.Add(jen.Return(jen.Lit(""), jen.Nil()))
			}),
	)
	return
}

func (body *RequestBody) UnmarshalYAML(value *yaml.Node) error {
	switch value.ShortTag() {
	case "!!str":
		switch value.Value {
		case "binary_data":
			body.BinaryData = true
		default:
			return fmt.Errorf("unknown request body type: %s", value.Value)
		}
		return nil
	case "!!map":
		switch value.Content[0].Value {
		case "json":
			return value.Content[1].Decode(&body.Json)
		case "form_urlencoded":
			return value.Content[1].Decode(&body.FormUrlencoded)
		case "multipart_form_data":
			return value.Content[1].Decode(&body.MultipartFormData)
		default:
			return fmt.Errorf("unknown request body type: %s", value.Content[0].Value)
		}
	default:
		return fmt.Errorf("unknown request body type: %s", value.ShortTag())
	}
}
