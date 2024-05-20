//go:build 1.12
// +build 1.12

package configfile

import "os"

func userHomeDir() (string, error) {
	return os.UserHomeDir()
}
