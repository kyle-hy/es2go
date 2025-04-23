package generator

import (
	"slices"
)

// 全局常量
const (
	MaxCombine = 4 // 随机组合字段的最大个数
)

// FuncTplData 预处理生产的函数模板需要的信息
type FuncTplData struct {
	Name    string // 函数名称
	Comment string // 函数注释
	Params  string // 参数列表
	Query   string // 查找条件
}

// DetailTplData 生成详情的模板数据
type DetailTplData struct {
	PackageName   string         // 代码包名
	StructName    string         // 模型结构体名称
	StructComment string         // 模型结构体注释
	IndexName     string         // es索引名称(表名)
	Fields        []*FieldInfo   // es相关字段信息
	FuncDatas     []*FuncTplData // 预处理生产的函数模板需要的信息
}

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
	EsFieldPath   string // es的字段名(嵌套)的访问路径
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

// limitCombination 检查某个组合是否满足类型限制
func limitCombination(comb []*FieldInfo, typeLimit map[string]int) bool {
	count := make(map[string]int)
	for _, item := range comb {
		tm := getTypeMapping(item.EsFieldType)
		count[tm]++
		if typeLimit[tm] != 0 && count[tm] > typeLimit[tm] {
			return false
		}
	}
	return true
}

// LimitCombineFilter 过滤出满足类型组合限制的组合
// typeLimit 整数则最大出现次数，负数则不允许出现
func LimitCombineFilter(combs [][]*FieldInfo, typeLimit map[string]int) [][]*FieldInfo {
	filterout := [][]*FieldInfo{}
	for _, comb := range combs {
		if limitCombination(comb, typeLimit) {
			filterout = append(filterout, comb)
		}
	}
	return filterout
}

// mustCombination 检查是否包含必须的类型
func mustCombination(comb []*FieldInfo, mustTypes []string) bool {
	for _, t := range mustTypes {
		for _, f := range comb {
			if getTypeMapping(f.EsFieldType) == t {
				return true
			}
		}
	}
	return false
}

// FieldFilterByTypes 根据类型拆分过滤字段
func FieldFilterByTypes(comb []*FieldInfo, mustTypes []string) (types []*FieldInfo, other []*FieldInfo) {
	for _, f := range comb {
		if slices.Contains(mustTypes, getTypeMapping(f.EsFieldType)) {
			types = append(types, f)
		} else {
			other = append(other, f)
		}
	}
	return
}

// MustCombineFilter 过滤出满足必须包含类型的组合
func MustCombineFilter(combs [][]*FieldInfo, mustTypes []string) [][]*FieldInfo {
	filterout := [][]*FieldInfo{}
	for _, comb := range combs {
		if mustCombination(comb, mustTypes) {
			filterout = append(filterout, comb)
		}
	}
	return filterout
}

// mustCombineField 检查是否包含必须的es字段
func mustCombineField(comb []*FieldInfo, mustFields []string) bool {
	for _, t := range mustFields {
		found := false
		for _, f := range comb {
			if f.EsFieldPath == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// MustCombineField 过滤出满足必须包含es字段的组合
func MustCombineField(combs [][]*FieldInfo, mustFields []string) [][]*FieldInfo {
	filterout := [][]*FieldInfo{}
	for _, comb := range combs {
		if mustCombineField(comb, mustFields) {
			filterout = append(filterout, comb)
		}
	}
	return filterout
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

func getTypeMapping(esType string) string {
	switch esType {
	case "dense_vector", "sparse_vector", "vector": // 向量类
		return TypeVector
	case "text", "wildcard", "constant_keyword", "match_only_text": // 字符串类
		return TypeText
	case "keyword": // 不做分词的字符串
		return TypeKeyword

	// 数值类
	case "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "unsigned_long":
		return TypeNumber

	case "date", "date_nanos": // 日期类
		return TypeDate

	case "boolean": // 布尔类
		return TypeBoolean

	// 范围类（特殊处理）
	case "integer_range", "float_range", "long_range", "double_range", "date_range":
		return TypeRange

	case "ip": // IP 地址类
		return TypeIP

	case "geo_point", "geo_shape": // 地理类
		return TypeGeo

	case "object", "nested", "flattened", "join": // 嵌套结构类
		return TypeObject

	case "binary", "token_count", "murmur3", "version": // 特殊类
		return TypeSpecial

	default:
		return TypeOther
	}
}

// TypeMapping 类型映射
func TypeMapping(field *FieldInfo) []string {
	types := []string{}
	if field.EsFieldType == "text" && field.FieldsKeyword == "keyword" {
		// 明确带 keyword 子字段
		types = append(types, TypeTextKeyword)
	}

	esType := field.EsFieldType
	types = append(types, getTypeMapping(esType))
	return types
}
