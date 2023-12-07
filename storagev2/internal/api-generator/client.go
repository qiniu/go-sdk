package main

import (
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
)

type (
	ApiDetailedDescription struct {
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
		Name, Documentation    string
		apiDetailedDescription *ApiDetailedDescription
	}
)

func (description *ApiDetailedDescription) generateSubPackages(group *jen.Group, _ CodeGeneratorOptions) (err error) {
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
	packageName := PackageNameApis + "/" + options.Name
	innerStructName := "inner" + strcase.ToCamel(options.Name) + "Request"
	reexportedRequestStructName := strcase.ToCamel(options.Name) + "Request"
	reexportedResponseStructName := strcase.ToCamel(options.Name) + "Response"
	group.Add(jen.Type().Id(innerStructName).Qual(packageName, "Request"))
	var getBucketNameGenerated bool
	if getBucketNameGenerated, err = description.generateGetBucketNameFunc(group, innerStructName); err != nil {
		return
	} else if err = description.generateBuildFunc(group, innerStructName); err != nil {
		return
	} else if err = description.addJsonMarshalerUnmarshaler(group, innerStructName, packageName, "Request"); err != nil {
		return
	} else if err = description.Request.generateGetAccessKeyFunc(group, innerStructName); err != nil {
		return
	}
	group.Add(jen.Type().Id(reexportedRequestStructName).Op("=").Qual(packageName, "Request"))
	group.Add(jen.Type().Id(reexportedResponseStructName).Op("=").Qual(packageName, "Response"))
	if options.Documentation != "" {
		group.Add(jen.Comment(options.Documentation))
	}
	group.Add(
		jen.Func().
			Params(jen.Id("storage").Op("*").Id("Storage")).
			Id(strcase.ToCamel(options.Name)).
			Params(
				jen.Id("ctx").Qual("context", "Context"),
				jen.Id("request").Op("*").Id(reexportedRequestStructName),
				jen.Id("options").Op("*").Id("Options"),
			).
			Params(
				jen.Id("response").Op("*").Id(reexportedResponseStructName),
				jen.Id("err").Error(),
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
					group.Add(jen.Id("pathSegments").Op("=").AppendFunc(func(group *jen.Group) {
						group.Add(jen.Id("pathSegments"))
						for _, pathSegment := range description.getPathSuffixSegments() {
							group.Add(jen.Lit(pathSegment))
						}
					}))
				}
				group.Add(jen.Id("path").Op(":=").Lit("/").Op("+").Qual("strings", "Join").Call(jen.Id("pathSegments"), jen.Lit("/")))
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
					}
				}
				method, err := description.Method.ToString()
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
											Call(jen.Id("innerRequest").Dot("Body")),
									)
								}
							}
						}),
				)
				group.Add(jen.Var().Id("queryer").Qual(PackageNameRegion, "BucketRegionsQueryer"))
				group.Add(
					jen.If(
						jen.Id("storage").Dot("client").Dot("GetRegions").Call().Op("==").Nil().Op("&&").
							Id("storage").Dot("client").Dot("GetEndpoints").Call().Op("==").Nil()).
						BlockFunc(func(group *jen.Group) {
							group.Add(jen.Id("queryer").Op("=").Id("storage").Dot("client").Dot("GetBucketQueryer").Call())
							group.Add(
								jen.If(jen.Id("queryer").Op("==").Nil()).BlockFunc(func(group *jen.Group) {
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
										group.Add(jen.Var().Id("err").Error())
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
											jen.Id("queryerOptions").
												Op(":=").
												Qual(PackageNameRegion, "BucketRegionsQueryerOptions").
												ValuesFunc(func(group *jen.Group) {
													group.Add(jen.Id("UseInsecureProtocol").Op(":").Id("storage").Dot("client").Dot("UseInsecureProtocol").Call())
													group.Add(jen.Id("HostFreezeDuration").Op(":").Id("storage").Dot("client").Dot("GetHostFreezeDuration").Call())
													group.Add(jen.Id("Client").Op(":").Id("storage").Dot("client").Dot("GetClient").Call())
												}),
										)
										group.Add(
											jen.If(
												jen.Id("hostRetryConfig").Op(":=").Id("storage").Dot("client").Dot("GetHostRetryConfig").Call(),
												jen.Id("hostRetryConfig").Op("!=").Nil(),
											).BlockFunc(func(group *jen.Group) {
												group.Id("queryerOptions").Dot("RetryMax").Op("=").Id("hostRetryConfig").Dot("RetryMax")
											}),
										)
										group.Add(
											jen.If(
												jen.List(jen.Id("queryer"), jen.Err()).
													Op("=").
													Qual(PackageNameRegion, "NewBucketRegionsQueryer").
													Call(jen.Id("bucketHosts"), jen.Op("&").Id("queryerOptions")),
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
						group.Add(jen.Id("bucketName").Op(":=").Id("options").Dot("OverwrittenBucketName"))
						group.Add(jen.Var().Id("accessKey").String())
						group.Add(jen.Var().Err().Error())
						if getBucketNameGenerated {
							group.Add(
								jen.If(jen.Id("bucketName").Op("==").Lit("")).BlockFunc(func(group *jen.Group) {
									group.Add(
										jen.If(
											jen.List(jen.Id("bucketName"), jen.Err()).Op("=").Id("innerRequest").Dot("getBucketName").Call(jen.Id("ctx")),
											jen.Err().Op("!=").Nil(),
										).BlockFunc(func(group *jen.Group) {
											group.Add(jen.Return(jen.Nil(), jen.Err()))
										}),
									)
								}),
							)
						}
						group.Add(
							jen.If(
								jen.List(jen.Id("accessKey"), jen.Err()).Op("=").Id("innerRequest").Dot("getAccessKey").Call(jen.Id("ctx")),
								jen.Err().Op("!=").Nil(),
							).BlockFunc(func(group *jen.Group) {
								group.Return(jen.Nil(), jen.Err())
							}).Else().
								If(jen.Id("accessKey").Op("==").Lit("")).
								BlockFunc(func(group *jen.Group) {
									group.Add(jen.If(
										jen.Id("credentialsProvider").Op(":=").Id("storage").Dot("client").Dot("GetCredentials").Call(),
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
							jen.Var().Id("respBody").Id(reexportedResponseStructName),
						)
						group.Add(
							jen.If(
								jen.List(jen.Id("_"), jen.Err()).
									Op(":=").
									Id("storage").
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
								Id("storage").
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
							Id("storage").
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
