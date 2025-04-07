package generator

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
	Meta       Meta                `json:"meta"` // 元数据，用于字段注释说明
	Properties map[string]Property `json:"properties,omitempty"`
}

// ElasticsearchMapping .
type ElasticsearchMapping struct {
	Mappings Mappings `json:"mappings"`
}

/******************* 模板渲染所需的信息 ******************/

// FieldInfo es的mapping字段信息
type FieldInfo struct {
	FieldName     string // golang的字段名称
	FieldType     string // golang的字段类型
	EsFieldType   string // es的mapping字段类型
	EsNestedField string // es的嵌套字段的访问路径
	JSONName      string // json字段标签
	FieldComment  string // 字段注释
	FieldsKeyword string // text字段keyword子字段
}

// EsModelInfo ES库表模型的信息
type EsModelInfo struct {
	PackageName   string       // 包名
	InitClassName string       // 封装的类型名称
	StructName    string       // 模型结构体名称
	StructComment string       // 模型结构体注释
	Fields        []*FieldInfo // 字段信息
}
