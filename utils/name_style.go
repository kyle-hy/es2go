package utils

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ToFirstLower 将驼峰模式的首字母小写
func ToFirstLower(s string) string {
	return strings.ToLower(s[:1]) + s[1:]
}

// ToCamelCase 下划线转驼峰模式,首字母小写
func ToCamelCase(s string) string {
	caser := cases.Title(language.Und) // or: `language.English`
	parts := strings.Split(s, "_")
	for i, part := range parts {
		parts[i] = caser.String(part)
	}
	parts[0] = strings.ToLower(parts[0])
	return strings.Join(parts, "")
}

// ToPascalCase 下划线转驼峰模式，首字母大写
func ToPascalCase(s string) string {
	caser := cases.Title(language.Und)
	parts := strings.Split(s, "_")
	for i, part := range parts {
		parts[i] = caser.String(part)
	}
	return strings.Join(parts, "")
}
