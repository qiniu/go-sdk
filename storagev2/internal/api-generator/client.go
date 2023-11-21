package main

import (
	"strings"

	"github.com/dave/jennifer/jen"
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

	CodeGenerator interface {
		Generate(group *jen.Group, options CodeGeneratorOptions) error
	}

	CodeGeneratorAliaser interface {
		CodeGenerator
		GenerateAliasesFor(group *jen.Group, structName, fieldName string) error
	}

	CodeGeneratorOptions struct {
		Name, Documentation    string
		apiDetailedDescription *ApiDetailedDescription
	}
)

func (description *ApiDetailedDescription) Generate(group *jen.Group, _ CodeGeneratorOptions) error {
	if pp := description.Request.PathParams; pp != nil {
		if err := pp.Generate(group, CodeGeneratorOptions{
			Name:                   "RequestPath",
			Documentation:          "调用 API 所用的路径参数",
			apiDetailedDescription: description,
		}); err != nil {
			return err
		} else if err = pp.GenerateAliasesFor(group, "Request", "Path"); err != nil {
			return err
		}
	}
	if queryNames := description.Request.QueryNames; queryNames != nil {
		if err := queryNames.Generate(group, CodeGeneratorOptions{
			Name:                   "RequestQuery",
			Documentation:          "调用 API 所用的 URL 查询参数",
			apiDetailedDescription: description,
		}); err != nil {
			return err
		} else if err = queryNames.GenerateAliasesFor(group, "Request", "Query"); err != nil {
			return err
		}
	}
	if headerNames := description.Request.HeaderNames; headerNames != nil {
		if err := headerNames.Generate(group, CodeGeneratorOptions{
			Name:                   "RequestHeaders",
			Documentation:          "调用 API 所用的 HTTP 头参数",
			apiDetailedDescription: description,
		}); err != nil {
			return err
		} else if err = headerNames.GenerateAliasesFor(group, "Request", "Headers"); err != nil {
			return err
		}
	}
	if body := description.Request.Body; body != nil {
		var codeGenerator CodeGeneratorAliaser
		if json := body.Json; json != nil {
			codeGenerator = json
		} else if formUrlencoded := body.FormUrlencoded; formUrlencoded != nil {
			codeGenerator = formUrlencoded
		} else if multipartFormData := body.MultipartFormData; multipartFormData != nil {
			codeGenerator = multipartFormData
		}
		if codeGenerator != nil {
			if err := codeGenerator.Generate(group, CodeGeneratorOptions{
				Name:                   "RequestBody",
				Documentation:          "调用 API 所用的请求体",
				apiDetailedDescription: description,
			}); err != nil {
				return err
			}
			if err := codeGenerator.GenerateAliasesFor(group, "Request", "Body"); err != nil {
				return err
			}
		}
	}
	if body := description.Response.Body; body != nil {
		if json := body.Json; json != nil {
			if err := json.Generate(group, CodeGeneratorOptions{
				Name:                   "ResponseBody",
				Documentation:          "获取 API 所用的响应体参数",
				apiDetailedDescription: description,
			}); err != nil {
				return err
			}
			if err := json.GenerateAliasesFor(group, "Response", "Body"); err != nil {
				return err
			}
		}
	}

	if err := description.Request.Generate(group, CodeGeneratorOptions{
		Name:                   "Request",
		Documentation:          "调用 API 所用的请求",
		apiDetailedDescription: description,
	}); err != nil {
		return err
	}
	if err := description.Response.Generate(group, CodeGeneratorOptions{
		Name:                   "Response",
		Documentation:          "获取 API 所用的响应",
		apiDetailedDescription: description,
	}); err != nil {
		return err
	}
	return nil
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
