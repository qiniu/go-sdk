package main

import (
	"path/filepath"
	"strings"
)

func extractApiSpecName(name string) string {
	baseName := filepath.Base(name)
	if index := strings.Index(baseName, "."); index >= 0 {
		baseName = baseName[:index]
	}
	return baseName
}
