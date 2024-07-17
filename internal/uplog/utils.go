package uplog

import "unicode/utf8"

const maxFieldValueLength = 1024

func truncate(s string, l int) string {
	if len(s) <= l {
		return s
	} else {
		i := l
		for ; i < len(s) && !utf8.ValidString(s[:i]); i++ {
		}
		return s[:i]
	}
}
