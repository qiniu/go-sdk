package main

import (
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
)

func extractApiSpecName(name string) string {
	baseName := filepath.Base(name)
	if index := strings.Index(baseName, "."); index >= 0 {
		baseName = baseName[:index]
	}
	return baseName
}

func makeGetterMethodName(fieldName string) string {
	fieldName = strcase.ToCamel(fieldName)
	if strings.HasPrefix(fieldName, "Is") {
		return fieldName
	}
	return "Get" + fieldName
}

func makeSetterMethodName(fieldName string) string {
	fieldName = strcase.ToCamel(fieldName)
	fieldName = strings.TrimPrefix(fieldName, "Is")
	return "Set" + fieldName
}
