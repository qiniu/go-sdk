package main

import (
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
)

type (
	ApiDetailedDescription struct {
		CamelCaseName string                 `yaml:"camel_case_name,omitempty"`
		SnakeCaseName string                 `yaml:"snake_case_name,omitempty"`
		Method        MethodName             `yaml:"method,omitempty"`
		ServiceNames  []ServiceName          `yaml:"service_names,omitempty"`
		Documentation string                 `yaml:"documentation,omitempty"`
		Command       string                 `yaml:"command,omitempty"`
		BasePath      string                 `yaml:"base_path,omitempty"`
		PathSuffix    string                 `yaml:"path_suffix,omitempty"`
		Request       ApiRequestDescription  `yaml:"request,omitempty"`
		Response      ApiResponseDescription `yaml:"response,omitempty"`
	}

	CodeGeneratorOptions struct {
		Name, Documentation          string
		CamelCaseName, SnakeCaseName string
		apiDetailedDescription       *ApiDetailedDescription
	}
)

func (options *CodeGeneratorOptions) camelCaseName() string {
	if options.CamelCaseName != "" {
		return options.CamelCaseName
	}
	return strcase.ToCamel(options.Name)
}

func (description *ApiDetailedDescription) generateSubPackages(group *jen.Group, _ CodeGeneratorOptions) (err error) {
	if body := description.Response.Body; body != nil {
		if bodyJson := body.Json; bodyJson != nil && bodyJson.Any {
			description.Request.responseTypeRequired = true
		}
	}
	if err = description.Request.generate(group, CodeGeneratorOptions{
		Name:                   "Request",
		Documentation:          "调用 API 所用的请求",
		apiDetailedDescription: description,
	}); err != nil {
		return
	}
	if err = description.Response.generate(group, CodeGeneratorOptions{
		Name:                   "Response",
		Documentation:          "获取 API 所用的响应",
		apiDetailedDescription: description,
	}); err != nil {
		return
	}
	return
}

func (description *ApiDetailedDescription) generatePackage(group *jen.Group, options CodeGeneratorOptions) (err error) {
	packageName := flags.ApiPackagePath + "/" + options.Name
	innerStructName := "inner" + options.camelCaseName() + "Request"
	reexportedRequestStructName := options.camelCaseName() + "Request"
	reexportedResponseStructName := options.camelCaseName() + "Response"
	group.Add(jen.Type().Id(innerStructName).Qual(packageName, "Request"))
	var getBucketNameGenerated, getObjectNameGenerated bool
	if getBucketNameGenerated, err = description.generateGetBucketNameFunc(group, innerStructName); err != nil {
		return
	} else if getObjectNameGenerated, err = description.generateGetObjectNameFunc(group, innerStructName); err != nil {
		return
	} else if err = description.generateBuildFunc(group, innerStructName); err != nil {
		return
	}
	if body := description.Request.Body; body != nil && body.Json != nil {
		if err = description.addJsonMarshalerUnmarshaler(group, innerStructName, packageName, "Request"); err != nil {
			return
		}
	}
	if isStorageAPIs() && !description.isBucketService() {
		if err = description.Request.generateGetAccessKeyFunc(group, innerStructName); err != nil {
			return
		}
	}
	group.Add(jen.Type().Id(reexportedRequestStructName).Op("=").Qual(packageName, "Request"))
	group.Add(jen.Type().Id(reexportedResponseStructName).Op("=").Qual(packageName, "Response"))
	if options.Documentation != "" {
		group.Add(jen.Comment(options.Documentation))
	}

	structName := strcase.ToCamel(flags.StructName)
	fieldName := strcase.ToLowerCamel(structName)

	group.Add(
		jen.Func().
			Params(jen.Id(fieldName).Op("*").Id(structName)).
			Id(options.camelCaseName()).
			Params(
				jen.Id("ctx").Qual("context", "Context"),
				jen.Id("request").Op("*").Id(reexportedRequestStructName),
				jen.Id("options").Op("*").Id("Options"),
			).
			Params(
				jen.Op("*").Id(reexportedResponseStructName),
				jen.Error(),
			).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.If(jen.Id("options").Op("==").Nil()).BlockFunc(func(group *jen.Group) {
					group.Id("options").Op("=").Op("&").Id("Options").Values()
				}))
				group.Add(jen.Id("innerRequest").Op(":=").Parens(jen.Op("*").Id(innerStructName)).Parens(jen.Id("request")))
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
				switch description.Request.Authorization.ToAuthorization() {
				case AuthorizationQbox, AuthorizationQiniu:
					group.Add(jen.If(
						jen.Id("innerRequest").Dot("Credentials").Op("==").Nil().Op("&&").
							Id(fieldName).Dot("client").Dot("GetCredentials").Call().Op("==").Nil()).
						BlockFunc(func(group *jen.Group) {
							group.Add(jen.Return(
								jen.Nil(),
								jen.Qual(PackageNameErrors, "MissingRequiredFieldError").
									ValuesFunc(func(group *jen.Group) {
										group.Add(jen.Id("Name").Op(":").Lit("Credentials"))
									}),
							))
						}))
				case AuthorizationUpToken:
					group.Add(jen.If(jen.Id("innerRequest").Dot("UpToken").Op("==").Nil()).
						BlockFunc(func(group *jen.Group) {
							group.Add(jen.Return(
								jen.Nil(),
								jen.Qual(PackageNameErrors, "MissingRequiredFieldError").
									ValuesFunc(func(group *jen.Group) {
										group.Add(jen.Id("Name").Op(":").Lit("UpToken"))
									}),
							))
						}))
				}
				if description.Request.HeaderNames != nil {
					group.Add(
						jen.List(jen.Id("headers"), jen.Err()).Op(":=").Id("innerRequest").Dot("buildHeaders").Call(),
					)
					group.Add(
						jen.If(
							jen.Err().Op("!=").Nil(),
						).BlockFunc(func(group *jen.Group) {
							group.Add(jen.Return(jen.Nil(), jen.Err()))
						}),
					)
				}
				guessPathSegmentsCount := 0
				if description.BasePath != "" {
					guessPathSegmentsCount += len(description.getBasePathSegments())
				}
				if description.PathSuffix != "" {
					guessPathSegmentsCount += len(description.getPathSuffixSegments())
				}
				if pp := description.Request.PathParams; pp != nil {
					for _, namedPathParam := range pp.Named {
						guessPathSegmentsCount += 1
						if namedPathParam.PathSegment != "" {
							guessPathSegmentsCount += 1
						}
					}
				}
				if guessPathSegmentsCount > 0 {
					group.Add(jen.Id("pathSegments").Op(":=").Make(jen.Index().String(), jen.Lit(0), jen.Lit(guessPathSegmentsCount)))
				} else {
					group.Add(jen.Id("pathSegments").Op(":=").Make(jen.Index().String(), jen.Lit(0)))
				}
				if description.BasePath != "" {
					if basePathSegments := description.getBasePathSegments(); len(basePathSegments) > 0 {
						group.Add(jen.Id("pathSegments").Op("=").AppendFunc(func(group *jen.Group) {
							group.Add(jen.Id("pathSegments"))
							for _, pathSegment := range basePathSegments {
								group.Add(jen.Lit(pathSegment))
							}
						}))
					}
				}
				if description.Request.PathParams != nil {
					group.Add(
						jen.If(
							jen.List(jen.Id("segments"), jen.Err()).
								Op(":=").
								Id("innerRequest").
								Dot("buildPath").
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
					if suffixSegments := description.getPathSuffixSegments(); len(suffixSegments) > 0 {
						group.Add(jen.Id("pathSegments").Op("=").AppendFunc(func(group *jen.Group) {
							group.Add(jen.Id("pathSegments"))
							for _, pathSegment := range suffixSegments {
								group.Add(jen.Lit(pathSegment))
							}
						}))
					}
				}
				group.Add(jen.Id("path").Op(":=").Lit("/").Op("+").Qual("strings", "Join").Call(jen.Id("pathSegments"), jen.Lit("/")))
				if description.PathSuffix == "/" {
					group.Add(jen.Id("path").Op("+=").Lit("/"))
				}
				if description.Command != "" {
					group.Add(jen.Id("rawQuery").Op(":=").Lit(description.Command + "&"))
				} else {
					group.Add(jen.Var().Id("rawQuery").String())
				}
				if description.Request.QueryNames != nil {
					group.Add(
						jen.If(
							jen.List(jen.Id("query"), jen.Err()).Op(":=").Id("innerRequest").Dot("buildQuery").Call(),
							jen.Err().Op("!=").Nil(),
						).BlockFunc(func(group *jen.Group) {
							group.Add(jen.Return(jen.Nil(), jen.Err()))
						}).Else().BlockFunc(func(group *jen.Group) {
							group.Id("rawQuery").Op("+=").Id("query").Dot("Encode").Call()
						}),
					)
				}
				if requestBody := description.Request.Body; requestBody != nil {
					if json := requestBody.Json; json != nil {
						group.Add(
							jen.List(jen.Id("body"), jen.Err()).
								Op(":=").
								Qual(PackageNameHTTPClient, "GetJsonRequestBody").
								Call(jen.Op("&").Id("innerRequest")),
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
								Id("innerRequest").
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
								Id("innerRequest").
								Dot("build").
								Call(),
						)
						group.Add(
							jen.If(jen.Err().Op("!=").Nil()).
								BlockFunc(func(group *jen.Group) {
									group.Add(jen.Return(jen.Nil(), jen.Err()))
								}),
						)
					} else if requestBody.BinaryData {
						group.Add(
							jen.Id("body").Op(":=").Id("innerRequest").Dot("Body"),
						)
						group.Add(
							jen.If(jen.Id("body").Op("==").Nil()).
								BlockFunc(func(group *jen.Group) {
									group.Add(jen.Return(
										jen.Nil(),
										jen.Qual(PackageNameErrors, "MissingRequiredFieldError").
											ValuesFunc(func(group *jen.Group) {
												group.Add(jen.Id("Name").Op(":").Lit("Body"))
											}),
									))
								}),
						)
					}
				}
				if isStorageAPIs() {
					group.Add(jen.Id("bucketName").Op(":=").Id("options").Dot("OverwrittenBucketName"))
					if getBucketNameGenerated {
						group.Add(
							jen.If(jen.Id("bucketName").Op("==").Lit("")).BlockFunc(func(group *jen.Group) {
								group.Add(jen.Var().Id("err").Error())
								group.Add(
									jen.If(
										jen.List(jen.Id("bucketName"), jen.Err()).Op("=").Id("innerRequest").Dot("getBucketName").Call(jen.Id("ctx")),
										jen.Err().Op("!=").Nil(),
									).BlockFunc(func(group *jen.Group) {
										group.Return(jen.Nil(), jen.Err())
									}),
								)
							}),
						)
					}
				}
				if isStorageAPIs() && getObjectNameGenerated {
					group.Add(jen.Id("objectName").Op(":=").Id("innerRequest").Dot("getObjectName").Call())
				}

				var (
					bucketNameVar jen.Code = jen.Lit("")
					objectNameVar jen.Code = jen.Lit("")
				)
				if isStorageAPIs() {
					bucketNameVar = jen.Id("bucketName")
					if getObjectNameGenerated {
						objectNameVar = jen.Id("objectName")
					}
				}

				var getUpTokenFunc jen.Code
				switch description.Request.Authorization.ToAuthorization() {
				case AuthorizationQbox, AuthorizationQiniu:
					getUpTokenFunc = jen.Func().Params().Params(jen.String(), jen.Error()).BlockFunc(func(group *jen.Group) {
						group.Add(
							jen.Id("credentials").
								Op(":=").
								Id("innerRequest").
								Dot("Credentials"))
						group.Add(jen.If(jen.Id("credentials").Op("==").Nil()).BlockFunc(func(group *jen.Group) {
							group.Add(
								jen.Id("credentials").
									Op("=").
									Id(fieldName).
									Dot("client").
									Dot("GetCredentials").
									Call())
						}))
						group.Add(
							jen.List(jen.Id("putPolicy"), jen.Err()).
								Op(":=").
								Qual(PackageNameUpToken, "NewPutPolicy").
								Call(bucketNameVar, jen.Qual("time", "Now").Call().Dot("Add").Call(jen.Qual("time", "Hour"))))
						group.Add(jen.If(jen.Err().Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
							group.Return(jen.List(jen.Lit(""), jen.Err()))
						}))
						group.Add(
							jen.Return(
								jen.Qual(PackageNameUpToken, "NewSigner").Call(jen.Id("putPolicy"), jen.Id("credentials")).Dot("GetUpToken").Call(jen.Id("ctx"))))
					})
				case AuthorizationUpToken:
					getUpTokenFunc = jen.Func().Params().Params(jen.String(), jen.Error()).BlockFunc(func(group *jen.Group) {
						group.Return(jen.Id("innerRequest").Dot("UpToken").Dot("GetUpToken").Call(jen.Id("ctx")))
					})
				case AuthorizationNone:
					getUpTokenFunc = jen.Nil()
				}

				group.Add(
					jen.List(jen.Id("uplogInterceptor"), jen.Err()).
						Op(":=").
						Qual(PackageNameUplog, "NewRequestUplog").Call(
						jen.Lit(strcase.ToLowerCamel(options.camelCaseName())),
						bucketNameVar,
						objectNameVar,
						getUpTokenFunc,
					),
				)
				group.Add(jen.If(jen.Err().Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
					group.Return(jen.Nil(), jen.Err())
				}))

				var method string
				method, err = description.Method.ToString()
				if err != nil {
					return
				}
				group.Add(
					jen.Id("req").
						Op(":=").
						Qual(PackageNameHTTPClient, "Request").
						ValuesFunc(func(group *jen.Group) {
							group.Add(jen.Id("Method").Op(":").Lit(method))
							group.Add(jen.Id("ServiceNames").Op(":").Id("serviceNames"))
							group.Add(jen.Id("Path").Op(":").Id("path"))
							group.Add(jen.Id("RawQuery").Op(":").Id("rawQuery"))
							group.Add(jen.Id("Endpoints").Op(":").Id("options").Dot("OverwrittenEndpoints"))
							group.Add(jen.Id("Region").Op(":").Id("options").Dot("OverwrittenRegion"))
							group.Add(jen.Id("Interceptors").Op(":").Index().Qual(PackageNameHTTPClient, "Interceptor").Values(jen.Id("uplogInterceptor")))
							if description.Request.HeaderNames != nil {
								group.Add(jen.Id("Header").Op(":").Id("headers"))
							}
							switch description.Request.Authorization.ToAuthorization() {
							case AuthorizationQbox:
								group.Add(jen.Id("AuthType").Op(":").Qual(PackageNameAuth, "TokenQBox"))
								group.Add(jen.Id("Credentials").Op(":").Id("innerRequest").Dot("Credentials"))
							case AuthorizationQiniu:
								group.Add(jen.Id("AuthType").Op(":").Qual(PackageNameAuth, "TokenQiniu"))
								group.Add(jen.Id("Credentials").Op(":").Id("innerRequest").Dot("Credentials"))
							case AuthorizationUpToken:
								group.Add(jen.Id("UpToken").Op(":").Id("innerRequest").Dot("UpToken"))
							}
							if body := description.Response.Body; body != nil {
								if json := body.Json; json != nil {
									group.Add(jen.Id("BufferResponse").Op(":").True())
								}
							}
							if requestBody := description.Request.Body; requestBody != nil {
								if json := requestBody.Json; json != nil {
									group.Add(jen.Id("RequestBody").Op(":").Id("body"))
								} else if formUrlencoded := requestBody.FormUrlencoded; formUrlencoded != nil {
									group.Add(
										jen.Id("RequestBody").
											Op(":").
											Qual(PackageNameHTTPClient, "GetFormRequestBody").
											Call(jen.Id("body")),
									)
								} else if multipartFormData := requestBody.MultipartFormData; multipartFormData != nil {
									group.Add(
										jen.Id("RequestBody").
											Op(":").
											Qual(PackageNameHTTPClient, "GetMultipartFormRequestBody").
											Call(jen.Id("body")),
									)
								} else if requestBody.BinaryData {
									group.Add(
										jen.Id("RequestBody").
											Op(":").
											Qual(PackageNameHTTPClient, "GetRequestBodyFromReadSeekCloser").
											Call(jen.Id("body")),
									)
								}
							}
							group.Add(jen.Id("OnRequestProgress").Op(":").Id("options").Dot("OnRequestProgress"))
						}),
				)
				group.Add(
					jen.If(
						jen.Id("options").Dot("OverwrittenEndpoints").Op("==").Nil().Op("&&").
							Id("options").Dot("OverwrittenRegion").Op("==").Nil().Op("&&").
							Id(fieldName).Dot("client").Dot("GetRegions").Call().Op("==").Nil()).
						BlockFunc(func(group *jen.Group) {
							group.Add(jen.Id("bucketHosts").Op(":=").Qual(PackageNameHTTPClient, "DefaultBucketHosts").Call())
							if description.isBucketService() {
								group.Add(
									jen.If(jen.Id("options").Dot("OverwrittenBucketHosts").Op("!=").Nil()).
										BlockFunc(func(group *jen.Group) {
											group.Add(jen.Id("req").Dot("Endpoints").Op("=").Id("options").Dot("OverwrittenBucketHosts"))
										}).
										Else().
										BlockFunc(func(group *jen.Group) {
											group.Add(jen.Id("req").Dot("Endpoints").Op("=").Id("bucketHosts"))
										}),
								)
							} else {
								ifBucketIsNotEmptyStmt := jen.If(jen.Id("bucketName").Op("!=").Lit("")).BlockFunc(func(group *jen.Group) {
									group.Add(jen.Id("query").Op(":=").Id(fieldName).Dot("client").Dot("GetBucketQuery").Call())
									group.Add(
										jen.If(jen.Id("query").Op("==").Nil()).BlockFunc(func(group *jen.Group) {
											group.Add(
												jen.If(jen.Id("options").Dot("OverwrittenBucketHosts").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
													group.Add(
														jen.If(
															jen.List(jen.Id("bucketHosts"), jen.Err()).
																Op("=").
																Id("options").
																Dot("OverwrittenBucketHosts").
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
												jen.Id("queryOptions").
													Op(":=").
													Qual(PackageNameRegion, "BucketRegionsQueryOptions").
													ValuesFunc(func(group *jen.Group) {
														group.Add(jen.Id("UseInsecureProtocol").Op(":").Id(fieldName).Dot("client").Dot("UseInsecureProtocol").Call())
														group.Add(jen.Id("AccelerateUploading").Op(":").Id(fieldName).Dot("client").Dot("AccelerateUploadingEnabled").Call())
														group.Add(jen.Id("HostFreezeDuration").Op(":").Id(fieldName).Dot("client").Dot("GetHostFreezeDuration").Call())
														group.Add(jen.Id("Resolver").Op(":").Id(fieldName).Dot("client").Dot("GetResolver").Call())
														group.Add(jen.Id("Chooser").Op(":").Id(fieldName).Dot("client").Dot("GetChooser").Call())
														group.Add(jen.Id("BeforeResolve").Op(":").Id(fieldName).Dot("client").Dot("GetBeforeResolveCallback").Call())
														group.Add(jen.Id("AfterResolve").Op(":").Id(fieldName).Dot("client").Dot("GetAfterResolveCallback").Call())
														group.Add(jen.Id("ResolveError").Op(":").Id(fieldName).Dot("client").Dot("GetResolveErrorCallback").Call())
														group.Add(jen.Id("BeforeBackoff").Op(":").Id(fieldName).Dot("client").Dot("GetBeforeBackoffCallback").Call())
														group.Add(jen.Id("AfterBackoff").Op(":").Id(fieldName).Dot("client").Dot("GetAfterBackoffCallback").Call())
														group.Add(jen.Id("BeforeRequest").Op(":").Id(fieldName).Dot("client").Dot("GetBeforeRequestCallback").Call())
														group.Add(jen.Id("AfterResponse").Op(":").Id(fieldName).Dot("client").Dot("GetAfterResponseCallback").Call())
													}),
											)
											group.Add(
												jen.If(
													jen.Id("hostRetryConfig").Op(":=").Id(fieldName).Dot("client").Dot("GetHostRetryConfig").Call(),
													jen.Id("hostRetryConfig").Op("!=").Nil(),
												).BlockFunc(func(group *jen.Group) {
													group.Id("queryOptions").Dot("RetryMax").Op("=").Id("hostRetryConfig").Dot("RetryMax")
													group.Id("queryOptions").Dot("Backoff").Op("=").Id("hostRetryConfig").Dot("Backoff")
												}),
											)
											group.Add(
												jen.If(
													jen.List(jen.Id("query"), jen.Err()).
														Op("=").
														Qual(PackageNameRegion, "NewBucketRegionsQuery").
														Call(jen.Id("bucketHosts"), jen.Op("&").Id("queryOptions")),
													jen.Err().Op("!=").Nil(),
												).BlockFunc(func(group *jen.Group) {
													group.Add(jen.Return(jen.Nil(), jen.Err()))
												}),
											)
										}),
									)
									group.Add(
										jen.If(jen.Id("query").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
											group.Add(jen.Var().Id("accessKey").String())
											group.Add(jen.Var().Err().Error())
											group.Add(
												jen.If(
													jen.List(jen.Id("accessKey"), jen.Err()).Op("=").Id("innerRequest").Dot("getAccessKey").Call(jen.Id("ctx")),
													jen.Err().Op("!=").Nil(),
												).BlockFunc(func(group *jen.Group) {
													group.Return(jen.Nil(), jen.Err())
												}),
											)
											group.Add(
												jen.If(jen.Id("accessKey").Op("==").Lit("")).
													BlockFunc(func(group *jen.Group) {
														group.Add(jen.If(
															jen.Id("credentialsProvider").Op(":=").Id(fieldName).Dot("client").Dot("GetCredentials").Call(),
															jen.Id("credentialsProvider").Op("!=").Nil(),
														).BlockFunc(func(group *jen.Group) {
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
														}))
													}),
											)
											group.Add(
												jen.If(jen.Id("accessKey").Op("!=").Lit("")).
													BlockFunc(func(group *jen.Group) {
														group.Id("req").Dot("Region").Op("=").Id("query").Dot("Query").Call(jen.Id("accessKey"), jen.Id("bucketName"))
													}),
											)
										}),
									)
								})
								ifBucketIsEmptyStmt := jen.CustomFunc(jen.Options{Multi: true}, func(group *jen.Group) {
									group.Add(jen.Id("req").Dot("Region").Op("=").Id(fieldName).Dot("client").Dot("GetAllRegions").Call())
									group.Add(
										jen.If(jen.Id("req").Dot("Region").Op("==").Nil()).BlockFunc(func(group *jen.Group) {
											group.Add(
												jen.If(jen.Id("options").Dot("OverwrittenBucketHosts").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
													group.Add(
														jen.If(
															jen.List(jen.Id("bucketHosts"), jen.Err()).
																Op("=").
																Id("options").
																Dot("OverwrittenBucketHosts").
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
												jen.Id("allRegionsOptions").
													Op(":=").
													Qual(PackageNameRegion, "AllRegionsProviderOptions").
													ValuesFunc(func(group *jen.Group) {
														group.Add(jen.Id("UseInsecureProtocol").Op(":").Id(fieldName).Dot("client").Dot("UseInsecureProtocol").Call())
														group.Add(jen.Id("HostFreezeDuration").Op(":").Id(fieldName).Dot("client").Dot("GetHostFreezeDuration").Call())
														group.Add(jen.Id("Resolver").Op(":").Id(fieldName).Dot("client").Dot("GetResolver").Call())
														group.Add(jen.Id("Chooser").Op(":").Id(fieldName).Dot("client").Dot("GetChooser").Call())
														group.Add(jen.Id("BeforeSign").Op(":").Id(fieldName).Dot("client").Dot("GetBeforeSignCallback").Call())
														group.Add(jen.Id("AfterSign").Op(":").Id(fieldName).Dot("client").Dot("GetAfterSignCallback").Call())
														group.Add(jen.Id("SignError").Op(":").Id(fieldName).Dot("client").Dot("GetSignErrorCallback").Call())
														group.Add(jen.Id("BeforeResolve").Op(":").Id(fieldName).Dot("client").Dot("GetBeforeResolveCallback").Call())
														group.Add(jen.Id("AfterResolve").Op(":").Id(fieldName).Dot("client").Dot("GetAfterResolveCallback").Call())
														group.Add(jen.Id("ResolveError").Op(":").Id(fieldName).Dot("client").Dot("GetResolveErrorCallback").Call())
														group.Add(jen.Id("BeforeBackoff").Op(":").Id(fieldName).Dot("client").Dot("GetBeforeBackoffCallback").Call())
														group.Add(jen.Id("AfterBackoff").Op(":").Id(fieldName).Dot("client").Dot("GetAfterBackoffCallback").Call())
														group.Add(jen.Id("BeforeRequest").Op(":").Id(fieldName).Dot("client").Dot("GetBeforeRequestCallback").Call())
														group.Add(jen.Id("AfterResponse").Op(":").Id(fieldName).Dot("client").Dot("GetAfterResponseCallback").Call())
													}),
											)
											group.Add(
												jen.If(
													jen.Id("hostRetryConfig").Op(":=").Id(fieldName).Dot("client").Dot("GetHostRetryConfig").Call(),
													jen.Id("hostRetryConfig").Op("!=").Nil(),
												).BlockFunc(func(group *jen.Group) {
													group.Id("allRegionsOptions").Dot("RetryMax").Op("=").Id("hostRetryConfig").Dot("RetryMax")
													group.Id("allRegionsOptions").Dot("Backoff").Op("=").Id("hostRetryConfig").Dot("Backoff")
												}),
											)
											group.Add(jen.Id("credentials").Op(":=").Id("innerRequest").Dot("Credentials"))
											group.Add(jen.If(jen.Id("credentials").Op("==").Nil()).BlockFunc(func(group *jen.Group) {
												group.Add(jen.Id("credentials").Op("=").Id(fieldName).Dot("client").Dot("GetCredentials").Call())
											}))
											group.Add(
												jen.If(
													jen.List(jen.Id("req").Dot("Region"), jen.Err()).
														Op("=").
														Qual(PackageNameRegion, "NewAllRegionsProvider").
														Call(jen.Id("credentials"), jen.Id("bucketHosts"), jen.Op("&").Id("allRegionsOptions")),
													jen.Err().Op("!=").Nil(),
												).BlockFunc(func(group *jen.Group) {
													group.Add(jen.Return(jen.Nil(), jen.Err()))
												}),
											)
										}),
									)
								})
								if !isStorageAPIs() {
									group.Add(ifBucketIsEmptyStmt)
								} else if description.isApiService() {
									group.Add(ifBucketIsNotEmptyStmt).Else().Block(ifBucketIsEmptyStmt)
								} else {
									group.Add(ifBucketIsNotEmptyStmt)
								}
							}
						}),
				)
				if description.Request.Authorization.ToAuthorization() == AuthorizationNone {
					group.Add(jen.Id("ctx").Op("=").Qual(PackageNameHTTPClient, "WithoutSignature").Call(jen.Id("ctx")))
				}
				if body := description.Response.Body; body != nil {
					if json := body.Json; json != nil {
						if description.Request.responseTypeRequired {
							group.Add(jen.Id("respBody").Op(":=").Id(reexportedResponseStructName).Values(jen.Id("Body").Op(":").Id("innerRequest").Dot("ResponseBody")))
						} else {
							group.Add(jen.Var().Id("respBody").Id(reexportedResponseStructName))
						}
						group.Add(
							jen.If(
								jen.Err().
									Op(":=").
									Id(fieldName).
									Dot("client").
									Dot("DoAndAcceptJSON").
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
						group.Add(jen.Return(jen.Op("&").Id("respBody"), jen.Nil()))
					} else if body.BinaryDataStream {
						group.Add(
							jen.List(jen.Id("resp"), jen.Err()).
								Op(":=").
								Id(fieldName).
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
									group.Return(jen.Nil(), jen.Err())
								}),
						)
						group.Add(
							jen.Return(
								jen.Op("&").
									Id(reexportedResponseStructName).
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
							Id(fieldName).
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
								group.Return(jen.Nil(), jen.Err())
							}),
					)
					group.Add(jen.Return(jen.Op("&").Id(reexportedResponseStructName).Values(), jen.Id("resp").Dot("Body").Dot("Close").Call()))
				}
			}),
	)
	return
}

func (description *ApiDetailedDescription) generateGetBucketNameFunc(group *jen.Group, structName string) (ok bool, err error) {
	if pp := description.Request.PathParams; pp != nil {
		if ok, err = pp.addGetBucketNameFunc(group, structName); err != nil || ok {
			return
		}
	}
	if queryNames := description.Request.QueryNames; queryNames != nil {
		if ok, err = queryNames.addGetBucketNameFunc(group, structName); err != nil || ok {
			return
		}
	}
	if headerNames := description.Request.HeaderNames; headerNames != nil {
		if ok, err = headerNames.addGetBucketNameFunc(group, structName); err != nil || ok {
			return
		}
	}
	if authorization := description.Request.Authorization; authorization != nil {
		if ok, err = authorization.addGetBucketNameFunc(group, structName); err != nil || ok {
			return
		}
	}
	if body := description.Request.Body; body != nil {
		if json := body.Json; json != nil {
			if jsonStruct := json.Struct; jsonStruct != nil {
				if ok, err = jsonStruct.addGetBucketNameFunc(group, structName); err != nil || ok {
					return
				}
			}
		} else if formUrlencoded := body.FormUrlencoded; formUrlencoded != nil {
			if ok, err = formUrlencoded.addGetBucketNameFunc(group, structName); err != nil || ok {
				return
			}
		} else if multipartFormData := body.MultipartFormData; multipartFormData != nil {
			if ok, err = multipartFormData.addGetBucketNameFunc(group, structName); err != nil || ok {
				return
			}
		}
	}
	return
}

func (description *ApiDetailedDescription) generateGetObjectNameFunc(group *jen.Group, structName string) (ok bool, err error) {
	if pp := description.Request.PathParams; pp != nil {
		if ok, err = pp.addGetObjectNameFunc(group, structName); err != nil || ok {
			return
		}
	}
	if queryNames := description.Request.QueryNames; queryNames != nil {
		if ok, err = queryNames.addGetObjectNameFunc(group, structName); err != nil || ok {
			return
		}
	}
	if headerNames := description.Request.HeaderNames; headerNames != nil {
		if ok, err = headerNames.addGetObjectNameFunc(group, structName); err != nil || ok {
			return
		}
	}
	if authorization := description.Request.Authorization; authorization != nil {
		if ok, err = authorization.addGetObjectNameFunc(group, structName); err != nil || ok {
			return
		}
	}
	if body := description.Request.Body; body != nil {
		if json := body.Json; json != nil {
			if jsonStruct := json.Struct; jsonStruct != nil {
				if ok, err = jsonStruct.addGetObjectNameFunc(group, structName); err != nil || ok {
					return
				}
			}
		} else if formUrlencoded := body.FormUrlencoded; formUrlencoded != nil {
			if ok, err = formUrlencoded.addGetObjectNameFunc(group, structName); err != nil || ok {
				return
			}
		} else if multipartFormData := body.MultipartFormData; multipartFormData != nil {
			if ok, err = multipartFormData.addGetObjectNameFunc(group, structName); err != nil || ok {
				return
			}
		}
	}
	return
}

func (description *ApiDetailedDescription) generateBuildFunc(group *jen.Group, structName string) (err error) {
	if pp := description.Request.PathParams; pp != nil {
		if err = pp.addBuildFunc(group, structName); err != nil {
			return
		}
	}
	if queryNames := description.Request.QueryNames; queryNames != nil {
		if err = queryNames.addBuildFunc(group, structName); err != nil {
			return
		}
	}
	if headerNames := description.Request.HeaderNames; headerNames != nil {
		if err = headerNames.addBuildFunc(group, structName); err != nil {
			return
		}
	}
	if body := description.Request.Body; body != nil {
		if formUrlencoded := body.FormUrlencoded; formUrlencoded != nil {
			if err = formUrlencoded.addBuildFunc(group, structName); err != nil {
				return
			}
		} else if multipartFormData := body.MultipartFormData; multipartFormData != nil {
			if err = multipartFormData.addBuildFunc(group, structName); err != nil {
				return
			}
		}
	}
	return
}

func (description *ApiDetailedDescription) addJsonMarshalerUnmarshaler(group *jen.Group, structName, actualPackageName, actualStructName string) (err error) {
	group.Add(
		jen.Func().
			Params(jen.Id("j").Op("*").Id(structName)).
			Id("MarshalJSON").
			Params().
			Params(jen.Index().Byte(), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Return(jen.Qual("encoding/json", "Marshal").Call(jen.Parens(jen.Op("*").Qual(actualPackageName, actualStructName)).Parens(jen.Id("j"))))
			}),
	)
	group.Add(
		jen.Func().
			Params(jen.Id("j").Op("*").Id(structName)).
			Id("UnmarshalJSON").
			Params(jen.Id("data").Index().Byte()).
			Params(jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Return(jen.Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Parens(jen.Op("*").Qual(actualPackageName, actualStructName)).Parens(jen.Id("j"))))
			}),
	)
	return
}

func (description *ApiDetailedDescription) getBasePathSegments() []string {
	basePath := strings.TrimPrefix(description.BasePath, "/")
	basePath = strings.TrimSuffix(basePath, "/")
	segments := strings.Split(basePath, "/")
	newSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment != "" {
			newSegments = append(newSegments, segment)
		}
	}
	return newSegments
}

func (description *ApiDetailedDescription) getPathSuffixSegments() []string {
	pathSuffix := strings.TrimPrefix(description.PathSuffix, "/")
	pathSuffix = strings.TrimSuffix(pathSuffix, "/")
	segments := strings.Split(pathSuffix, "/")
	newSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment != "" {
			newSegments = append(newSegments, segment)
		}
	}
	return newSegments
}

func (description *ApiDetailedDescription) isBucketService() bool {
	for _, service := range description.ServiceNames {
		if service == ServiceNameBucket {
			return true
		}
	}
	return false
}

func (description *ApiDetailedDescription) isApiService() bool {
	for _, service := range description.ServiceNames {
		if service == ServiceNameApi {
			return true
		}
	}
	return false
}
