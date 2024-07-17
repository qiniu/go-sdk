package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
	goflags "github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v3"
)

const (
	PackageNameHTTPClient  = "github.com/qiniu/go-sdk/v7/storagev2/http_client"
	PackageNameAuth        = "github.com/qiniu/go-sdk/v7/auth"
	PackageNameCredentials = "github.com/qiniu/go-sdk/v7/storagev2/credentials"
	PackageNameRegion      = "github.com/qiniu/go-sdk/v7/storagev2/region"
	PackageNameUpToken     = "github.com/qiniu/go-sdk/v7/storagev2/uptoken"
	PackageNameErrors      = "github.com/qiniu/go-sdk/v7/storagev2/errors"
	PackageNameUplog       = "github.com/qiniu/go-sdk/v7/internal/uplog"
	PackageNameInternalIo  = "github.com/qiniu/go-sdk/v7/internal/io"
)

var flags struct {
	ApiSpecsPaths  []string `long:"api-specs" required:"true"`
	OutputDirPath  string   `long:"output" required:"true"`
	ApiPackagePath string   `long:"api-package" required:"true"`
	StructName     string   `long:"struct-name" required:"true"`
}

func main() {
	_, err := goflags.ParseArgs(&flags, os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	os.RemoveAll(flags.OutputDirPath)
	for _, apiSpecsPath := range flags.ApiSpecsPaths {
		entries, err := ioutil.ReadDir(apiSpecsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read directory %s: %s\n", apiSpecsPath, err)
			os.Exit(1)
		}

		for _, entry := range entries {
			apiSpecName := extractApiSpecName(entry.Name())
			generatedApiDirPath := filepath.Join(flags.OutputDirPath, apiSpecName)
			if err = os.MkdirAll(generatedApiDirPath, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create directory %s: %s\n", generatedApiDirPath, err)
				os.Exit(1)
			}
			if err = writeGolangPackages(apiSpecName, filepath.Join(apiSpecsPath, entry.Name()), flags.OutputDirPath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write go package %s: %s\n", apiSpecName, err)
				os.Exit(1)
			}
		}
	}
	if err = writeApiClient(flags.OutputDirPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write api client: %s\n", err)
		os.Exit(1)
	}
}

func writeGolangPackages(apiSpecName, apiSpecPath, generatedDirPath string) (err error) {
	generatedApiDirPath := filepath.Join(generatedDirPath, apiSpecName)
	apiSpecFile, err := os.Open(apiSpecPath)
	if err != nil {
		return
	}
	defer apiSpecFile.Close()

	var apiSpec ApiDetailedDescription
	decoder := yaml.NewDecoder(apiSpecFile)
	decoder.KnownFields(true)
	if err = decoder.Decode(&apiSpec); err != nil {
		return
	}
	if err = apiSpecFile.Close(); err != nil {
		return
	}

	if err = writeSubPackage(apiSpecName, generatedApiDirPath, &apiSpec); err != nil {
		return
	}
	return writeApiPackage(apiSpecName, generatedDirPath, &apiSpec)
}

func writeSubPackage(apiSpecName, generatedDirPath string, apiSpec *ApiDetailedDescription) error {
	packageFile := jen.NewFile(apiSpecName)
	packageFile.HeaderComment("THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!")
	packageFile.PackageComment(apiSpec.Documentation)
	if err := apiSpec.generateSubPackages(packageFile.Group, CodeGeneratorOptions{
		Name:          apiSpecName,
		CamelCaseName: apiSpec.CamelCaseName,
		SnakeCaseName: apiSpec.SnakeCaseName,
		Documentation: apiSpec.Documentation,
	}); err != nil {
		return err
	}
	return packageFile.Save(filepath.Join(generatedDirPath, "api.go"))
}

func writeApiPackage(apiSpecName, generatedDirPath string, apiSpec *ApiDetailedDescription) error {
	apisPackageFile := jen.NewFile("apis")
	apisPackageFile.HeaderComment("THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!")
	if err := apiSpec.generatePackage(apisPackageFile.Group, CodeGeneratorOptions{
		Name:          apiSpecName,
		CamelCaseName: apiSpec.CamelCaseName,
		SnakeCaseName: apiSpec.SnakeCaseName,
		Documentation: apiSpec.Documentation,
	}); err != nil {
		return err
	}
	return apisPackageFile.Save(filepath.Join(generatedDirPath, "api_"+apiSpecName+".go"))
}

func writeApiClient(generatedDirPath string) error {
	apiPackageFile := jen.NewFile("apis")
	apiPackageFile.HeaderComment("THIS FILE IS GENERATED BY api-generator, DO NOT EDIT DIRECTLY!")
	generateApiClient(apiPackageFile.Group)
	return apiPackageFile.Save(filepath.Join(generatedDirPath, "apis.go"))
}

func generateApiClient(group *jen.Group) {
	structName := strcase.ToCamel(flags.StructName)
	group.Add(jen.Comment("API 客户端"))
	group.Add(
		jen.Type().Id(structName).StructFunc(func(group *jen.Group) {
			group.Add(jen.Id("client").Op("*").Qual(PackageNameHTTPClient, "Client"))
		}),
	)
	group.Add(jen.Comment("创建 API 客户端"))
	group.Add(
		jen.Func().
			Id("New" + structName).
			Params(jen.Id("options").Op("*").Qual(PackageNameHTTPClient, "Options")).
			Params(jen.Op("*").Id(structName)).
			BlockFunc(func(group *jen.Group) {
				group.Return(jen.Op("&").Id(structName).ValuesFunc(func(group *jen.Group) {
					group.Add(jen.Id("client").Op(":").Qual(PackageNameHTTPClient, "NewClient").Call(jen.Id("options")))
				}))
			}),
	)

	group.Add(jen.Comment("API 客户端选项"))
	group.Add(
		jen.Type().Id("Options").StructFunc(func(group *jen.Group) {
			group.Add(jen.Id("OverwrittenBucketHosts").Qual(PackageNameRegion, "EndpointsProvider"))
			group.Add(jen.Id("OverwrittenBucketName").String())
			group.Add(jen.Id("OverwrittenEndpoints").Qual(PackageNameRegion, "EndpointsProvider"))
			group.Add(jen.Id("OverwrittenRegion").Qual(PackageNameRegion, "RegionsProvider"))
			group.Add(jen.Id("OnRequestProgress").Func().Params(
				jen.Uint64(),
				jen.Uint64(),
			))
		}),
	)
}

func isStorageAPIs() bool {
	structName := strcase.ToCamel(flags.StructName)
	return structName == "Storage"
}
