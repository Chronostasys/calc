package helper

import "strings"

func SplitLast(str, sep string) []string {
	mainStr := str
	if strings.Contains(str, "<") {
		mainStr = strings.Split(mainStr, "<")[0]
	}
	idx := strings.LastIndex(mainStr, sep)
	if idx == -1 {
		return []string{str}
	}
	last := str[idx+1:]
	first := str[:idx]
	return []string{first, last}
}

func LastBlock(str string) string {
	idx := strings.LastIndex(str, ".")
	if idx == -1 {
		return str
	}
	return str[idx+1:]
}

func Trim(str string) string {
	return strings.Trim(str, "%*\"")
}
