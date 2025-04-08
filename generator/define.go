package generator

import "fmt"

/***************** es mapping 相关 **************************/

// Keyword 属性的子类型
type Keyword struct {
	Type string `json:"type,omitempty"`
}

// Fields 属性子字段
type Fields struct {
	Keyword Keyword `json:"keyword"`
}

// Meta 属性的注释说明
type Meta struct {
	Comment string `json:"comment,omitempty"`
}

// Property 字段属性
type Property struct {
	Type       string              `json:"type,omitempty"`
	Meta       Meta                `json:"meta"` // 元数据，用于字段注释说明
	Fields     Fields              `json:"fields"`
	Properties map[string]Property `json:"properties,omitempty"`
}

// Mappings .
type Mappings struct {
	Meta       Meta                `json:"_meta"` // 使用保留字段，用于库表注释说明
	Properties map[string]Property `json:"properties,omitempty"`
}

// ElasticsearchMapping .
type ElasticsearchMapping struct {
	Mappings Mappings `json:"mappings"`
}

/******************* 模板渲染所需的信息 ******************/

// FieldInfo es的mapping字段信息
type FieldInfo struct {
	FieldName     string // go的字段名称
	FieldType     string // go的字段类型
	JSONName      string // go的字段json字段标签,es字段原名
	FieldComment  string // go的字段注释
	FieldsKeyword string // go的字段text子字段keyword
	EsFieldType   string // es的mapping字段类型
	EsFieldPath   string // es的字段(嵌套)的访问路径
}

// EsModelInfo ES库表模型的信息
type EsModelInfo struct {
	PackageName   string       // go的包名
	InitClassName string       // go自定义封装的类型名称
	StructName    string       // go的模型结构体名称
	StructComment string       // go的模型结构体注释
	IndexName     string       // es的索引(表)名称
	Fields        []*FieldInfo // es相关字段信息
}

// GroupFieldsByType 按照数据类型划分字段
func GroupFieldsByType(fields []*FieldInfo) map[string][]*FieldInfo {
	grouped := make(map[string][]*FieldInfo)
	for _, f := range fields {
		typ := TypeMapping(f)
		grouped[typ] = append(grouped[typ], f)
	}
	return grouped
}

// TypeMapping 类型映射
func TypeMapping(field *FieldInfo) string {
	if field.EsFieldType == "text" && field.FieldsKeyword == "keyword" {
		fmt.Println(field)
		// 明确带 keyword 子字段
		return "text.keyword"
	}

	esType := field.EsFieldType
	switch esType {
	// 字符串类
	case "text", "wildcard", "constant_keyword", "match_only_text":
		return "text"
	case "keyword":
		return "keyword"

	// 数值类
	case "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "unsigned_long":
		return "number"

	// 日期类
	case "date", "date_nanos":
		return "date"

	// 布尔类
	case "boolean":
		return "boolean"

	// 范围类（特殊处理）
	case "integer_range", "float_range", "long_range", "double_range", "date_range":
		return "range"

	// IP 地址类
	case "ip":
		return "ip"

	// 地理类
	case "geo_point", "geo_shape":
		return "geo"

	// 嵌套结构类
	case "object", "nested", "flattened", "join":
		return "object"

	// 特殊类
	case "binary", "token_count", "murmur3", "version":
		return "special"

	default:
		return "other"
	}
}
