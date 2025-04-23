package utils

import (
	"strings"
)

// FuncComment 提取标准注释的首行不含函数名的注释
func FuncComment(comment string) string {
	// 拆成行
	lines := strings.Split(comment, "\n")
	if len(lines) > 0 {
		// 取第一行，去掉前缀 //
		firstLine := strings.TrimSpace(strings.TrimPrefix(lines[0], "//"))
		// 分割函数名和注释
		parts := strings.SplitN(firstLine, " ", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return ""
}

// FirstLine 提取首行内容
func FirstLine(comment string) string {
	// 拆成行
	lines := strings.Split(comment, "\n")
	if len(lines) > 0 {
		return strings.Replace(lines[0], "\"", "", -1)
	}
	return ""
}
