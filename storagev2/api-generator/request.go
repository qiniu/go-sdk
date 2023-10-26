package main

import (
	"fmt"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
	"gopkg.in/yaml.v3"
)

type (
	ApiRequestDescription struct {
		PathParams    *PathParams    `yaml:"path_params,omitempty"`
		HeaderNames   HeaderNames    `yaml:"header_names,omitempty"`
		QueryNames    QueryNames     `yaml:"query_names,omitempty"`
		Body          *RequestBody   `yaml:"body,omitempty"`
		Authorization *Authorization `yaml:"authorization,omitempty"`
		Idempotent    *Idempotent    `yaml:"idempotent,omitempty"`
	}

	RequestBody struct {
		Json              *JsonType
		FormUrlencoded    *FormUrlencodedRequestStruct
		MultipartFormData *MultipartFormFields
		BinaryData        bool
	}
)

func (request *ApiRequestDescription) Generate(group *jen.Group, opts CodeGeneratorOptions) error {
	if opts.Documentation != "" {
		group.Add(jen.Comment(opts.Documentation))
	}
	group.Add(
		jen.Type().
			Id(strcase.ToCamel(opts.Name)).
			StructFunc(func(group *jen.Group) {
				group.Add(jen.Id("BucketHosts").Qual("github.com/qiniu/go-sdk/v7/storagev2/region", "EndpointsProvider"))

				if pp := request.PathParams; pp != nil {
					group.Add(jen.Id("Path").Id("RequestPath"))
				}
				if names := request.QueryNames; names != nil {
					group.Add(jen.Id("Query").Id("RequestQuery"))
				}
				if names := request.HeaderNames; names != nil {
					group.Add(jen.Id("Headers").Id("RequestHeaders"))
				}
				if authorization := request.Authorization; authorization != nil {
					switch authorization.ToAuthorization() {
					case AuthorizationQbox, AuthorizationQiniu:
						group.Add(jen.Id("Credentials").Qual("github.com/qiniu/go-sdk/v7/storagev2/credentials", "CredentialsProvider"))
					case AuthorizationUpToken:
						group.Add(jen.Id("UpToken").Qual("github.com/qiniu/go-sdk/v7/storagev2/uptoken", "Provider"))
					}
				}
				if body := request.Body; body != nil {
					if body.Json != nil {
						group.Add(jen.Id("Body").Id("RequestBody"))
					} else if body.FormUrlencoded != nil {
						group.Add(jen.Id("Body").Id("RequestBody"))
					} else if body.MultipartFormData != nil {
						group.Add(jen.Id("Body").Id("RequestBody"))
					} else if body.BinaryData {
						group.Add(jen.Id("Body").Qual("github.com/qiniu/go-sdk/v7/internal/io", "ReadSeekCloser"))
					}
				}
			}),
	)
	group.Add(
		jen.Func().
			Params(jen.Id("request").Id(strcase.ToCamel(opts.Name))).
			Id("getBucketName").
			Params(jen.Id("ctx").Qual("context", "Context")).
			Params(jen.Id("string"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				if pp := request.PathParams; pp != nil {
					if field := pp.getServiceBucketField(); field.ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
						group.Add(
							jen.If(
								jen.List(jen.Id("bucketName"), jen.Err()).Op(":=").Id("request").Dot("Path").Dot("getBucketName").Call(),
								jen.Err().Op("!=").Nil().Op("||").Id("bucketName").Op("!=").Lit(""),
							).BlockFunc(func(group *jen.Group) {
								group.Add(jen.Return(jen.Id("bucketName"), jen.Err()))
							}),
						)
					}
				}
				if query := request.QueryNames; query != nil {
					if field := query.getServiceBucketField(); field.ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
						group.Add(
							jen.If(
								jen.List(jen.Id("bucketName"), jen.Err()).Op(":=").Id("request").Dot("Query").Dot("getBucketName").Call(),
								jen.Err().Op("!=").Nil().Op("||").Id("bucketName").Op("!=").Lit(""),
							).BlockFunc(func(group *jen.Group) {
								group.Add(jen.Return(jen.Id("bucketName"), jen.Err()))
							}),
						)
					}
				}
				if authorization := request.Authorization; authorization != nil {
					if authorization.ToAuthorization() == AuthorizationUpToken {
						group.Add(
							jen.If(jen.Id("request").Dot("UpToken").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
								group.Add(
									jen.If(
										jen.List(jen.Id("putPolicy"), jen.Err()).Op(":=").Id("request").Dot("UpToken").Dot("RetrievePutPolicy").Call(jen.Id("ctx")),
										jen.Err().Op("!=").Nil(),
									).BlockFunc(func(group *jen.Group) {
										group.Add(jen.Return(jen.Lit(""), jen.Err()))
									}).Else().BlockFunc(func(group *jen.Group) {
										group.Add(jen.Return(jen.Id("putPolicy").Dot("GetBucketName").Call()))
									}),
								)
							}),
						)
					}
				}
				if body := request.Body; body != nil {
					var (
						params               []jen.Code
						hasServiceBucketType = false
					)
					if json := body.Json; json != nil {
						if jsonStruct := json.Struct; jsonStruct != nil {
							if field := jsonStruct.getServiceBucketField(); field != nil {
								if field.ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
									hasServiceBucketType = true
								}
							}
						}
					} else if form := body.FormUrlencoded; form != nil {
						if field := form.getServiceBucketField(); field.ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
							hasServiceBucketType = true
						}
					} else if multipartForm := body.MultipartFormData; multipartForm != nil {
						if field := multipartForm.getServiceBucketField(); field.ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
							hasServiceBucketType = true
							params = append(params, jen.Id("ctx"))
						}
					}
					if hasServiceBucketType {
						group.Add(
							jen.If(
								jen.List(jen.Id("bucketName"), jen.Err()).Op(":=").Id("request").Dot("Body").Dot("getBucketName").Call(params...),
								jen.Err().Op("!=").Nil().Op("||").Id("bucketName").Op("!=").Lit(""),
							).BlockFunc(func(group *jen.Group) {
								group.Add(jen.Return(jen.Id("bucketName"), jen.Err()))
							}),
						)
					}
				}
				group.Add(jen.Return(jen.Lit(""), jen.Nil()))
			}),
	)
	group.Add(
		jen.Func().
			Params(jen.Id("request").Id(strcase.ToCamel(opts.Name))).
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
										group.Add(jen.Return(jen.Lit(""), jen.Err()))
									}).Else().BlockFunc(func(group *jen.Group) {
										group.Add(jen.Return(jen.Id("credentials").Dot("AccessKey"), jen.Nil()))
									}),
								)
							}),
						)
					case AuthorizationUpToken:
						group.Add(
							jen.If(jen.Id("request").Dot("UpToken").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
								group.Add(jen.Return(jen.Id("request").Dot("UpToken").Dot("RetrieveAccessKey").Call(jen.Id("ctx"))))
							}),
						)
					}
				}
				if body := request.Body; body != nil {
					if multipartForm := body.MultipartFormData; multipartForm != nil {
						if field := multipartForm.getServiceBucketField(); field.ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
							fieldName := "field" + strcase.ToCamel(field.FieldName)
							group.Add(
								jen.If(jen.Id("request").Dot("Body").Dot(fieldName).Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
									group.Add(
										jen.If(
											jen.List(jen.Id("accessKey"), jen.Err()).
												Op(":=").
												Id("request").
												Dot("Body").
												Dot(fieldName).
												Dot("RetrieveAccessKey").
												Call(jen.Id("ctx")),
											jen.Err().Op("!=").Nil(),
										).BlockFunc(func(group *jen.Group) {
											group.Add(jen.Return(jen.Lit(""), jen.Err()))
										}).Else().BlockFunc(func(group *jen.Group) {
											group.Add(jen.Return(jen.Id("accessKey"), jen.Nil()))
										}),
									)
								}),
							)
						}
					}
				}
				group.Add(jen.Return(jen.Lit(""), jen.Nil()))
			}),
	)
	return nil
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
