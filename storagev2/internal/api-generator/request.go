package main

import (
	"fmt"
	"strings"

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
	request.generateRequestStruct(group, opts)
	request.generateOverwriteBucketHostsFunc(group, opts)
	request.generateOverwriteBucketNameFunc(group, opts)
	request.generateSetAuthFunc(group, opts)
	request.generateGetBucketNameFunc(group, opts)
	request.generateGetAccessKeyFunc(group, opts)
	return request.generateSendFunc(group, opts)
}

func (request *ApiRequestDescription) generateRequestStruct(group *jen.Group, opts CodeGeneratorOptions) {
	if opts.Documentation != "" {
		group.Add(jen.Comment(opts.Documentation))
	}
	group.Add(
		jen.Type().
			Id(strcase.ToCamel(opts.Name)).
			StructFunc(func(group *jen.Group) {
				group.Add(jen.Id("overwrittenBucketHosts").Qual(PackageNameRegion, "EndpointsProvider"))
				group.Add(jen.Id("overwrittenBucketName").Id("string"))

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
						group.Add(jen.Id("credentials").Qual(PackageNameCredentials, "CredentialsProvider"))
					case AuthorizationUpToken:
						group.Add(jen.Id("upToken").Qual(PackageNameUpToken, "Provider"))
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
						group.Add(jen.Id("Body").Qual(PackageNameInternalIo, "ReadSeekCloser"))
					}
				}
			}),
	)
}

func (request *ApiRequestDescription) generateOverwriteBucketHostsFunc(group *jen.Group, opts CodeGeneratorOptions) {
	structName := strcase.ToCamel(opts.Name)
	group.Add(
		jen.Func().
			Params(jen.Id("request").Id(structName)).
			Id("OverwriteBucketHosts").
			Params(jen.Id("bucketHosts").Qual(PackageNameRegion, "EndpointsProvider")).
			Params(jen.Id(structName)).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.Id("request").Dot("overwrittenBucketHosts").Op("=").Id("bucketHosts"))
				group.Add(jen.Return(jen.Id("request")))
			}),
	)
}

func (request *ApiRequestDescription) generateOverwriteBucketNameFunc(group *jen.Group, opts CodeGeneratorOptions) {
	structName := strcase.ToCamel(opts.Name)
	group.Add(
		jen.Func().
			Params(jen.Id("request").Id(structName)).
			Id("OverwriteBucketName").
			Params(jen.Id("bucketName").String()).
			Params(jen.Id(structName)).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.Id("request").Dot("overwrittenBucketName").Op("=").Id("bucketName"))
				group.Add(jen.Return(jen.Id("request")))
			}),
	)
}

func (request *ApiRequestDescription) generateSetAuthFunc(group *jen.Group, opts CodeGeneratorOptions) {
	structName := strcase.ToCamel(opts.Name)
	if authorization := request.Authorization; authorization != nil {
		switch authorization.ToAuthorization() {
		case AuthorizationQbox, AuthorizationQiniu:
			group.Add(
				jen.Func().
					Params(jen.Id("request").Id(structName)).
					Id("SetCredentials").
					Params(jen.Id("credentials").Qual(PackageNameCredentials, "CredentialsProvider")).
					Params(jen.Id(structName)).
					BlockFunc(func(group *jen.Group) {
						group.Add(jen.Id("request").Dot("credentials").Op("=").Id("credentials"))
						group.Add(jen.Return(jen.Id("request")))
					}),
			)
		case AuthorizationUpToken:
			group.Add(
				jen.Func().
					Params(jen.Id("request").Id(structName)).
					Id("SetUpToken").
					Params(jen.Id("upToken").Qual(PackageNameUpToken, "Provider")).
					Params(jen.Id(structName)).
					BlockFunc(func(group *jen.Group) {
						group.Add(jen.Id("request").Dot("upToken").Op("=").Id("upToken"))
						group.Add(jen.Return(jen.Id("request")))
					}),
			)
		}
	}
}

func (request *ApiRequestDescription) generateGetBucketNameFunc(group *jen.Group, opts CodeGeneratorOptions) {
	group.Add(
		jen.Func().
			Params(jen.Id("request").Id(strcase.ToCamel(opts.Name))).
			Id("getBucketName").
			Params(jen.Id("ctx").Qual("context", "Context")).
			Params(jen.Id("string"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.If(jen.Id("request").Dot("overwrittenBucketName").Op("!=").Lit("")).BlockFunc(func(group *jen.Group) {
						group.Return(jen.Id("request").Dot("overwrittenBucketName"), jen.Nil())
					}),
				)
				if pp := request.PathParams; pp != nil {
					if field := pp.getServiceBucketField(); field.ServiceBucket.ToServiceBucketType() != ServiceBucketTypeNone {
						group.Add(
							jen.If(
								jen.List(jen.Id("bucketName"), jen.Err()).Op(":=").Id("request").Dot("Path").Dot("getBucketName").Call(),
								jen.Err().Op("!=").Nil().Op("||").Id("bucketName").Op("!=").Lit(""),
							).BlockFunc(func(group *jen.Group) {
								group.Return(jen.Id("bucketName"), jen.Err())
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
								group.Return(jen.Id("bucketName"), jen.Err())
							}),
						)
					}
				}
				if authorization := request.Authorization; authorization != nil {
					if authorization.ToAuthorization() == AuthorizationUpToken {
						group.Add(
							jen.If(jen.Id("request").Dot("upToken").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
								group.Add(
									jen.If(
										jen.List(jen.Id("putPolicy"), jen.Err()).Op(":=").Id("request").Dot("upToken").Dot("RetrievePutPolicy").Call(jen.Id("ctx")),
										jen.Err().Op("!=").Nil(),
									).BlockFunc(func(group *jen.Group) {
										group.Return(jen.Lit(""), jen.Err())
									}).Else().BlockFunc(func(group *jen.Group) {
										group.Return(jen.Id("putPolicy").Dot("GetBucketName").Call())
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
								group.Return(jen.Id("bucketName"), jen.Err())
							}),
						)
					}
				}
				group.Add(jen.Return(jen.Lit(""), jen.Nil()))
			}),
	)
}

func (request *ApiRequestDescription) generateGetAccessKeyFunc(group *jen.Group, opts CodeGeneratorOptions) {
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
							jen.If(jen.Id("request").Dot("credentials").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
								group.Add(
									jen.If(
										jen.List(jen.Id("credentials"), jen.Err()).
											Op(":=").
											Id("request").
											Dot("credentials").
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
							jen.If(jen.Id("request").Dot("upToken").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
								group.Return(jen.Id("request").Dot("upToken").Dot("RetrieveAccessKey").Call(jen.Id("ctx")))
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
									group.If(
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
									})
								}),
							)
						}
					}
				}
				group.Add(jen.Return(jen.Lit(""), jen.Nil()))
			}),
	)
}

func (request *ApiRequestDescription) generateSendFunc(group *jen.Group, opts CodeGeneratorOptions) (err error) {
	description := opts.apiDetailedDescription
	group.Add(
		jen.Func().
			Params(jen.Id("request").Id(strcase.ToCamel(opts.Name))).
			Id("Send").
			Params(
				jen.Id("ctx").Qual("context", "Context"),
				jen.Id("options").Op("*").Qual(PackageNameHttpClient, "HttpClientOptions"),
			).
			Params(jen.Op("*").Id("Response"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.Id("client").Op(":=").Qual(PackageNameHttpClient, "NewHttpClient").Call(jen.Id("options")))
				group.Add(
					jen.Id("serviceNames").
						Op(":=").
						Index().
						Qual(PackageNameRegion, "ServiceName").
						ValuesFunc(func(group *jen.Group) {
							for _, serviceName := range description.ServiceNames {
								if code, e := serviceName.ToServiceName(); e != nil {
									err = e
									return
								} else {
									group.Add(code)
								}
							}
						}),
				)
				group.Add(jen.Var().Id("pathSegments").Index().String())
				if description.BasePath != "" {
					group.Add(jen.Id("pathSegments").Op("=").AppendFunc(func(group *jen.Group) {
						group.Add(jen.Id("pathSegments"))
						for _, pathSegment := range description.getBasePathSegments() {
							group.Add(jen.Lit(pathSegment))
						}
					}))
				}
				if description.Request.PathParams != nil {
					group.Add(
						jen.If(
							jen.List(jen.Id("segments"), jen.Err()).
								Op(":=").
								Id("request").
								Dot("Path").
								Dot("build").
								Call(),
							jen.Err().Op("!=").Nil(),
						).BlockFunc(func(group *jen.Group) {
							group.Add(jen.Return(jen.Nil(), jen.Err()))
						}).Else().BlockFunc(func(group *jen.Group) {
							group.Add(jen.Id("pathSegments").Op("=").Append(
								jen.Id("pathSegments"),
								jen.Id("segments").Op("..."),
							))
						}),
					)
				}
				if description.PathSuffix != "" {
					group.Add(jen.Id("pathSegments").Op("=").AppendFunc(func(group *jen.Group) {
						group.Add(jen.Id("pathSegments"))
						for _, pathSegment := range description.getPathSuffixSegments() {
							group.Add(jen.Lit(pathSegment))
						}
					}))
				}
				group.Add(jen.Id("path").Op(":=").Lit("/").Op("+").Qual("strings", "Join").Call(jen.Id("pathSegments"), jen.Lit("/")))
				if description.Request.QueryNames != nil {
					group.Add(jen.List(jen.Id("query"), jen.Err()).Op(":=").Id("request").Dot("Query").Dot("build").Call())
					group.Add(
						jen.If(jen.Err().Op("!=").Nil()).
							BlockFunc(func(group *jen.Group) {
								group.Add(jen.Return(jen.Nil(), jen.Err()))
							}),
					)
				}

				if requestBody := description.Request.Body; requestBody != nil {
					if json := requestBody.Json; json != nil {
						if json.Struct != nil {
							group.Add(
								jen.If(
									jen.Err().
										Op(":=").
										Id("request").
										Dot("Body").
										Dot("validate").
										Call(),
									jen.Err().Op("!=").Nil(),
								).BlockFunc(func(group *jen.Group) {
									group.Add(jen.Return(jen.Nil(), jen.Err()))
								}),
							)
						}
						group.Add(
							jen.List(jen.Id("body"), jen.Err()).
								Op(":=").
								Qual(PackageNameHttpClient, "GetJsonRequestBody").
								Call(jen.Op("&").Id("request").Dot("Body")),
						)
						group.Add(
							jen.If(jen.Err().Op("!=").Nil()).
								BlockFunc(func(group *jen.Group) {
									group.Add(jen.Return(jen.Nil(), jen.Err()))
								}),
						)
					} else if multipartForm := requestBody.MultipartFormData; multipartForm != nil {
						group.Add(
							jen.List(jen.Id("body"), jen.Err()).
								Op(":=").
								Id("request").
								Dot("Body").
								Dot("build").
								Call(jen.Id("ctx")),
						)
						group.Add(
							jen.If(jen.Err().Op("!=").Nil()).
								BlockFunc(func(group *jen.Group) {
									group.Add(jen.Return(jen.Nil(), jen.Err()))
								}),
						)
					} else if form := requestBody.FormUrlencoded; form != nil {
						group.Add(
							jen.List(jen.Id("body"), jen.Err()).
								Op(":=").
								Id("request").
								Dot("Body").
								Dot("build").
								Call(),
						)
						group.Add(
							jen.If(jen.Err().Op("!=").Nil()).
								BlockFunc(func(group *jen.Group) {
									group.Add(jen.Return(jen.Nil(), jen.Err()))
								}),
						)
					}
				}
				group.Add(
					jen.Id("req").
						Op(":=").
						Qual(PackageNameHttpClient, "Request").
						ValuesFunc(func(group *jen.Group) {
							group.Add(jen.Id("Method").Op(":").Lit(strings.ToUpper(description.Method)))
							group.Add(jen.Id("ServiceNames").Op(":").Id("serviceNames"))
							group.Add(jen.Id("Path").Op(":").Id("path"))
							if description.Request.QueryNames != nil {
								group.Add(jen.Id("RawQuery").Op(":").Id("query").Dot("Encode").Call())
							}
							if description.Request.HeaderNames != nil {
								group.Add(jen.Id("Header").Op(":").Id("request").Dot("Headers").Dot("build").Call())
							}
							switch description.Request.Authorization.ToAuthorization() {
							case AuthorizationQbox:
								group.Add(jen.Id("AuthType").Op(":").Qual(PackageNameAuth, "TokenQBox"))
								group.Add(jen.Id("Credentials").Op(":").Id("request").Dot("credentials"))
							case AuthorizationQiniu:
								group.Add(jen.Id("AuthType").Op(":").Qual(PackageNameAuth, "TokenQiniu"))
								group.Add(jen.Id("Credentials").Op(":").Id("request").Dot("credentials"))
							case AuthorizationUpToken:
								group.Add(jen.Id("UpToken").Op(":").Id("request").Dot("upToken"))
							}
							if requestBody := description.Request.Body; requestBody != nil {
								if json := requestBody.Json; json != nil {
									group.Add(jen.Id("RequestBody").Op(":").Id("body"))
								} else if formUrlencoded := requestBody.FormUrlencoded; formUrlencoded != nil {
									group.Add(
										jen.Id("RequestBody").
											Op(":").
											Qual(PackageNameHttpClient, "GetFormRequestBody").
											Call(jen.Id("body")),
									)
								} else if multipartFormData := requestBody.MultipartFormData; multipartFormData != nil {
									group.Add(
										jen.Id("RequestBody").
											Op(":").
											Qual(PackageNameHttpClient, "GetMultipartFormRequestBody").
											Call(jen.Id("body")),
									)
								} else if requestBody.BinaryData {
									group.Add(
										jen.Id("RequestBody").
											Op(":").
											Qual(PackageNameHttpClient, "GetRequestBodyFromReadSeekCloser").
											Call(jen.Id("request").Dot("Body")),
									)
								}
							}
						}),
				)
				group.Add(jen.Var().Id("queryer").Qual(PackageNameRegion, "BucketRegionsQueryer"))
				group.Add(
					jen.If(
						jen.Id("client").Dot("GetRegions").Call().Op("==").Nil().Op("&&").
							Id("client").Dot("GetEndpoints").Call().Op("==").Nil()).
						BlockFunc(func(group *jen.Group) {
							group.Add(jen.Id("queryer").Op("=").Id("client").Dot("GetBucketQueryer").Call())
							group.Add(
								jen.If(jen.Id("queryer").Op("==").Nil()).BlockFunc(func(group *jen.Group) {
									group.Add(jen.Id("bucketHosts").Op(":=").Qual(PackageNameHttpClient, "DefaultBucketHosts").Call())
									if description.isBucketService() {
										group.Add(
											jen.If(jen.Id("request").Dot("overwrittenBucketHosts").Op("!=").Nil()).
												BlockFunc(func(group *jen.Group) {
													group.Add(jen.Id("req").Dot("Endpoints").Op("=").Id("request").Dot("overwrittenBucketHosts"))
												}).
												Else().
												BlockFunc(func(group *jen.Group) {
													group.Add(jen.Id("req").Dot("Endpoints").Op("=").Id("bucketHosts"))
												}),
										)
									} else {
										group.Add(jen.Var().Id("err").Error())
										group.Add(
											jen.If(jen.Id("request").Dot("overwrittenBucketHosts").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
												group.Add(
													jen.If(
														jen.List(jen.Id("bucketHosts"), jen.Err()).
															Op("=").
															Id("request").
															Dot("overwrittenBucketHosts").
															Dot("GetEndpoints").
															Call(jen.Id("ctx")),
														jen.Err().Op("!=").Nil(),
													).BlockFunc(func(group *jen.Group) {
														group.Add(jen.Return(jen.Nil(), jen.Err()))
													}),
												)
											}),
										)
										group.Add(
											jen.If(
												jen.List(jen.Id("queryer"), jen.Err()).
													Op("=").
													Qual(PackageNameRegion, "NewBucketRegionsQueryer").
													Call(jen.Id("bucketHosts"), jen.Nil()),
												jen.Err().Op("!=").Nil(),
											).BlockFunc(func(group *jen.Group) {
												group.Add(jen.Return(jen.Nil(), jen.Err()))
											}),
										)
									}
								}),
							)
						}),
				)
				group.Add(
					jen.If(jen.Id("queryer").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
						group.Add(
							jen.List(jen.Id("bucketName"), jen.Err()).Op(":=").Id("request").Dot("getBucketName").Call(jen.Id("ctx")),
						)
						group.Add(
							jen.If(jen.Err().Op("!=").Nil()).
								BlockFunc(func(group *jen.Group) {
									group.Add(jen.Return(jen.Nil(), jen.Err()))
								}),
						)
						group.Add(
							jen.List(jen.Id("accessKey"), jen.Err()).Op(":=").Id("request").Dot("getAccessKey").Call(jen.Id("ctx")),
						)
						group.Add(
							jen.If(jen.Err().Op("!=").Nil()).
								BlockFunc(func(group *jen.Group) {
									group.Return(jen.Nil(), jen.Err())
								}),
						)
						group.Add(
							jen.If(jen.Id("accessKey").Op("==").Lit("")).
								BlockFunc(func(group *jen.Group) {
									group.Add(jen.If(
										jen.Id("credentialsProvider").Op(":=").Id("client").Dot("GetCredentials").Call(),
										jen.Id("credentialsProvider").Op("!=").Nil(),
									)).BlockFunc(func(group *jen.Group) {
										group.If(
											jen.List(jen.Id("creds"), jen.Err()).
												Op(":=").
												Id("credentialsProvider").
												Dot("Get").
												Call(jen.Id("ctx")),
											jen.Err().Op("!=").Nil(),
										).BlockFunc(func(group *jen.Group) {
											group.Return(jen.Nil(), jen.Err())
										}).Else().
											If(jen.Id("creds").Op("!=").Nil()).
											BlockFunc(func(group *jen.Group) {
												group.Id("accessKey").Op("=").Id("creds").Dot("AccessKey")
											})
									})
								}),
						)
						group.Add(
							jen.If(jen.Id("accessKey").Op("!=").Lit("").Op("&&").Id("bucketName").Op("!=").Lit("")).
								BlockFunc(func(group *jen.Group) {
									group.Id("req").Dot("Region").Op("=").Id("queryer").Dot("Query").Call(jen.Id("accessKey"), jen.Id("bucketName"))
								}),
						)
					}),
				)
				if body := description.Response.Body; body != nil {
					if json := body.Json; json != nil {
						group.Add(
							jen.Var().Id("respBody").Id("ResponseBody"),
						)
						group.Add(
							jen.If(
								jen.List(jen.Id("_"), jen.Err()).
									Op(":=").
									Id("client").
									Dot("AcceptJson").
									Call(
										jen.Id("ctx"),
										jen.Op("&").Id("req"),
										jen.Op("&").Id("respBody"),
									),
								jen.Err().Op("!=").Nil(),
							).BlockFunc(func(group *jen.Group) {
								group.Return(jen.Nil(), jen.Err())
							}),
						)
						group.Add(
							jen.Return(
								jen.Op("&").
									Id("Response").
									Values(
										jen.Id("Body").
											Op(":").
											Id("respBody"),
									),
								jen.Nil(),
							),
						)
					} else if body.BinaryDataStream {
						group.Add(
							jen.List(jen.Id("resp"), jen.Err()).
								Op(":=").
								Id("client").
								Dot("Do").
								Call(
									jen.Id("ctx"),
									jen.Op("&").Id("req"),
								),
						)
						group.Add(
							jen.If(jen.Err().Op("!=").Nil()).
								BlockFunc(func(group *jen.Group) {
									group.Return(jen.Nil(), jen.Err())
								}),
						)
						group.Add(
							jen.Return(
								jen.Op("&").
									Id("Response").
									Values(jen.Id("Body").
										Op(":").
										Id("resp").
										Dot("Body")), jen.Nil(),
							),
						)
					}
				} else {
					group.Add(
						jen.List(jen.Id("resp"), jen.Err()).
							Op(":=").
							Id("client").
							Dot("Do").
							Call(
								jen.Id("ctx"),
								jen.Op("&").Id("req"),
							),
					)
					group.Add(
						jen.If(jen.Err().Op("!=").Nil()).
							BlockFunc(func(group *jen.Group) {
								group.Return(jen.Nil(), jen.Err())
							}),
					)
					group.Add(jen.Defer().Id("resp").Dot("Body").Dot("Close").Call())
					group.Add(jen.Return(jen.Op("&").Id("Response").Values(), jen.Nil()))
				}
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
