package main

import (
	"strings"

	"github.com/dave/jennifer/jen"
)

type (
	ApiDetailedDescription struct {
		Method        string                 `yaml:"method,omitempty"`
		ServiceNames  []ServiceName          `yaml:"service_names,omitempty"`
		Documentation string                 `yaml:"documentation,omitempty"`
		BasePath      string                 `yaml:"base_path,omitempty"`
		PathSuffix    string                 `yaml:"path_suffix,omitempty"`
		Request       ApiRequestDescription  `yaml:"request,omitempty"`
		Response      ApiResponseDescription `yaml:"response,omitempty"`
	}

	CodeGenerator interface {
		Generate(group *jen.Group, options CodeGeneratorOptions) error
	}

	CodeGeneratorOptions struct {
		Name, Documentation string
	}
)

func (description *ApiDetailedDescription) Generate(group *jen.Group, _ CodeGeneratorOptions) error {
	if pp := description.Request.PathParams; pp != nil {
		if err := pp.Generate(group, CodeGeneratorOptions{
			Name:          "RequestPath",
			Documentation: "调用 API 所用的路径参数",
		}); err != nil {
			return err
		}
	}
	if queryNames := description.Request.QueryNames; queryNames != nil {
		if err := queryNames.Generate(group, CodeGeneratorOptions{
			Name:          "RequestQuery",
			Documentation: "调用 API 所用的 URL 查询参数",
		}); err != nil {
			return err
		}
	}
	if headerNames := description.Request.HeaderNames; headerNames != nil {
		if err := headerNames.Generate(group, CodeGeneratorOptions{
			Name:          "RequestHeaders",
			Documentation: "调用 API 所用的 HTTP 头参数",
		}); err != nil {
			return err
		}
	}
	if body := description.Request.Body; body != nil {
		var codeGenerator CodeGenerator
		if json := body.Json; json != nil {
			codeGenerator = json
		} else if formUrlencoded := body.FormUrlencoded; formUrlencoded != nil {
			codeGenerator = formUrlencoded
		} else if multipartFormData := body.MultipartFormData; multipartFormData != nil {
			codeGenerator = multipartFormData
		}
		if codeGenerator != nil {
			if err := codeGenerator.Generate(group, CodeGeneratorOptions{
				Name:          "RequestBody",
				Documentation: "调用 API 所用的请求体",
			}); err != nil {
				return err
			}
		}
	}
	if body := description.Response.Body; body != nil {
		if json := body.Json; json != nil {
			if err := json.Generate(group, CodeGeneratorOptions{
				Name:          "ResponseBody",
				Documentation: "获取 API 所用的响应体参数",
			}); err != nil {
				return err
			}
		}
	}

	if err := description.Request.Generate(group, CodeGeneratorOptions{
		Name:          "Request",
		Documentation: "调用 API 所用的请求",
	}); err != nil {
		return err
	}
	if err := description.Response.Generate(group, CodeGeneratorOptions{
		Name:          "Response",
		Documentation: "获取 API 所用的响应",
	}); err != nil {
		return err
	}
	if err := description.generateClient(group); err != nil {
		return err
	}
	return nil
}

func (description *ApiDetailedDescription) generateClient(group *jen.Group) error {
	var err error
	group.Add(jen.Comment("API 调用客户端"))
	group.Add(
		jen.Type().
			Id("Client").
			Struct(
				jen.Id("client").Op("*").Qual(PackageNameHttpClient, "HttpClient"),
			),
	)
	group.Add(jen.Comment("创建 API 调用客户端"))
	group.Add(
		jen.Func().
			Id("NewClient").
			Params(jen.Id("options").Op("*").Qual(PackageNameHttpClient, "HttpClientOptions")).
			Params(jen.Op("*").Id("Client")).
			Block(
				jen.Id("client").Op(":=").Qual(PackageNameHttpClient, "NewHttpClient").Call(jen.Id("options")),
				jen.Return(jen.Op("&").Id("Client").Values(jen.Id("client").Op(":").Id("client"))),
			),
	)
	group.Add(
		jen.Func().
			Params(jen.Id("client").Op("*").Id("Client")).
			Id("Send").
			Params(
				jen.Id("ctx").Qual("context", "Context"),
				jen.Id("request").Op("*").Id("Request"),
			).
			Params(jen.Op("*").Id("Response"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(
					jen.Id("serviceNames").
						Op(":=").
						Index().
						Qual("github.com/qiniu/go-sdk/v7/storagev2/region", "ServiceName").
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
								Qual("github.com/qiniu/go-sdk/v7/storagev2/http_client", "GetJsonRequestBody").
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
						Qual("github.com/qiniu/go-sdk/v7/storagev2/http_client", "Request").
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
								group.Add(jen.Id("AuthType").Op(":").Qual("github.com/qiniu/go-sdk/v7/auth", "TokenQBox"))
								group.Add(jen.Id("Credentials").Op(":").Id("request").Dot("Credentials"))
							case AuthorizationQiniu:
								group.Add(jen.Id("AuthType").Op(":").Qual("github.com/qiniu/go-sdk/v7/auth", "TokenQiniu"))
								group.Add(jen.Id("Credentials").Op(":").Id("request").Dot("Credentials"))
							case AuthorizationUpToken:
								group.Add(jen.Id("UpToken").Op(":").Id("request").Dot("UpToken"))
							}
							if requestBody := description.Request.Body; requestBody != nil {
								if json := requestBody.Json; json != nil {
									group.Add(jen.Id("RequestBody").Op(":").Id("body"))
								} else if formUrlencoded := requestBody.FormUrlencoded; formUrlencoded != nil {
									group.Add(
										jen.Id("RequestBody").
											Op(":").
											Qual("github.com/qiniu/go-sdk/v7/storagev2/http_client", "GetFormRequestBody").
											Call(jen.Id("body")),
									)
								} else if multipartFormData := requestBody.MultipartFormData; multipartFormData != nil {
									group.Add(
										jen.Id("RequestBody").
											Op(":").
											Qual("github.com/qiniu/go-sdk/v7/storagev2/http_client", "GetMultipartFormRequestBody").
											Call(jen.Id("body")),
									)
								} else if requestBody.BinaryData {
									group.Add(
										jen.Id("RequestBody").
											Op(":").
											Qual("github.com/qiniu/go-sdk/v7/storagev2/http_client", "GetRequestBodyFromReadSeekCloser").
											Call(jen.Id("request").Dot("Body")),
									)
								}
							}
						}),
				)
				group.Add(jen.Var().Id("queryer").Qual("github.com/qiniu/go-sdk/v7/storagev2/region", "BucketRegionsQueryer"))
				group.Add(
					jen.If(
						jen.Id("client").Dot("client").Dot("GetRegions").Call().Op("==").Nil().Op("&&").
							Id("client").Dot("client").Dot("GetEndpoints").Call().Op("==").Nil()).
						BlockFunc(func(group *jen.Group) {
							group.Add(jen.Id("queryer").Op("=").Id("client").Dot("client").Dot("GetBucketQueryer").Call())
							group.Add(
								jen.If(jen.Id("queryer").Op("==").Nil()).BlockFunc(func(group *jen.Group) {
									group.Add(jen.Id("bucketHosts").Op(":=").Qual("github.com/qiniu/go-sdk/v7/storagev2/http_client", "DefaultBucketHosts").Call())
									if description.isBucketService() {
										group.Add(
											jen.If(jen.Id("request").Dot("BucketHosts").Op("!=").Nil()).
												BlockFunc(func(group *jen.Group) {
													group.Add(jen.Id("req").Dot("Endpoints").Op("=").Id("request").Dot("BucketHosts"))
												}).
												Else().
												BlockFunc(func(group *jen.Group) {
													group.Add(jen.Id("req").Dot("Endpoints").Op("=").Id("bucketHosts"))
												}),
										)
									} else {
										group.Add(jen.Var().Id("err").Error())
										group.Add(
											jen.If(jen.Id("request").Dot("BucketHosts").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
												group.Add(
													jen.If(
														jen.List(jen.Id("bucketHosts"), jen.Err()).
															Op("=").
															Id("request").
															Dot("BucketHosts").
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
													Qual("github.com/qiniu/go-sdk/v7/storagev2/region", "NewBucketRegionsQueryer").
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
									group.Add(jen.Return(jen.Nil(), jen.Err()))
								}),
						)
						group.Add(
							jen.If(jen.Id("accessKey").Op("!=").Lit("").Op("&&").Id("bucketName").Op("!=").Lit("")).
								BlockFunc(func(group *jen.Group) {
									group.Add(
										jen.Id("req").Dot("Region").Op("=").Id("queryer").Dot("Query").Call(jen.Id("accessKey"), jen.Id("bucketName")),
									)
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
									Dot("client").
									Dot("AcceptJson").
									Call(
										jen.Id("ctx"),
										jen.Op("&").Id("req"),
										jen.Op("&").Id("respBody"),
									),
								jen.Err().Op("!=").Nil(),
							).BlockFunc(func(group *jen.Group) {
								group.Add(jen.Return(jen.Nil(), jen.Err()))
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
								Dot("client").
								Dot("Do").
								Call(
									jen.Id("ctx"),
									jen.Op("&").Id("req"),
								),
						)
						group.Add(
							jen.If(jen.Err().Op("!=").Nil()).
								BlockFunc(func(group *jen.Group) {
									group.Add(jen.Return(jen.Nil(), jen.Err()))
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
							Dot("client").
							Dot("Do").
							Call(
								jen.Id("ctx"),
								jen.Op("&").Id("req"),
							),
					)
					group.Add(
						jen.If(jen.Err().Op("!=").Nil()).
							BlockFunc(func(group *jen.Group) {
								group.Add(jen.Return(jen.Nil(), jen.Err()))
							}),
					)
					group.Add(jen.Defer().Id("resp").Dot("Body").Dot("Close").Call())
					group.Add(jen.Return(jen.Op("&").Id("Response").Values(), jen.Nil()))
				}
			}),
	)
	return err
}

func (description *ApiDetailedDescription) getBasePathSegments() []string {
	basePath := strings.TrimPrefix(description.BasePath, "/")
	basePath = strings.TrimSuffix(basePath, "/")
	return strings.Split(basePath, "/")
}

func (description *ApiDetailedDescription) getPathSuffixSegments() []string {
	pathSuffix := strings.TrimPrefix(description.PathSuffix, "/")
	pathSuffix = strings.TrimSuffix(pathSuffix, "/")
	return strings.Split(pathSuffix, "/")
}

func (description *ApiDetailedDescription) isBucketService() bool {
	for _, service := range description.ServiceNames {
		if service == ServiceNameBucket {
			return true
		}
	}
	return false
}
