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
	JSONName      string // go的字段json字段标签
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
