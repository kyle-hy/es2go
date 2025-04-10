package generator

// 全局常量
const (
	MaxCombine = 5
)

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
		types := TypeMapping(f)
		for _, typ := range types {
			grouped[typ] = append(grouped[typ], f)
		}
	}
	return grouped
}

// 类型分组
const (
	TypeVector      = "vector"
	TypeText        = "text"
	TypeKeyword     = "keyword"
	TypeTextKeyword = "text.keyword"
	TypeNumber      = "number"
	TypeDate        = "date"
	TypeBoolean     = "boolean"
	TypeRange       = "range"
	TypeIP          = "ip"
	TypeGeo         = "geo"
	TypeObject      = "object"
	TypeSpecial     = "special"
	TypeOther       = "other"
)

// TypeMapping 类型映射
func TypeMapping(field *FieldInfo) []string {
	types := []string{}
	if field.EsFieldType == "text" && field.FieldsKeyword == "keyword" {
		// 明确带 keyword 子字段
		types = append(types, TypeTextKeyword)
	}

	esType := field.EsFieldType
	switch esType {
	case "dense_vector", "sparse_vector": // 向量类
		types = append(types, TypeVector)
		return types
	case "text", "wildcard", "constant_keyword", "match_only_text": // 字符串类
		types = append(types, TypeText)
		return types
	case "keyword": // 不做分词的字符串
		types = append(types, TypeKeyword)
		return types

	// 数值类
	case "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "unsigned_long":
		types = append(types, TypeNumber)
		return types

	case "date", "date_nanos": // 日期类
		types = append(types, TypeDate)
		return types

	case "boolean": // 布尔类
		types = append(types, TypeBoolean)
		return types

	// 范围类（特殊处理）
	case "integer_range", "float_range", "long_range", "double_range", "date_range":
		types = append(types, TypeRange)
		return types

	case "ip": // IP 地址类
		types = append(types, TypeIP)
		return types

	case "geo_point", "geo_shape": // 地理类
		types = append(types, TypeGeo)
		return types

	case "object", "nested", "flattened", "join": // 嵌套结构类
		types = append(types, TypeObject)
		return types

	case "binary", "token_count", "murmur3", "version": // 特殊类
		types = append(types, TypeSpecial)
		return types

	default:
		types = append(types, TypeOther)
		return types
	}
}
