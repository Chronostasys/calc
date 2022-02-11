package helper

import "strings"

func SplitLast(str, sep string) []string {
	idx := strings.LastIndex(str, sep)
	if idx == -1 {
		return []string{str}
	}
	last := str[idx+1:]
	first := str[:idx]
	return []string{first, last}
}
