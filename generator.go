package es2go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Keyword 属性的子类型
type Keyword struct {
	Type string `json:"type,omitempty"`
}

// Fields 属性子字段
type Fields struct {
	Keyword Keyword `json:"keyword,omitempty"`
}

// Meta 属性的注释说明
type Meta struct {
	Comment string `json:"comment,omitempty"`
}

// Property 字段属性
type Property struct {
	Type       string              `json:"type,omitempty"`
	Meta       Meta                `json:"meta,omitempty"` // 元数据，用于字段注释说明
	Fields     Fields              `json:"fields,omitempty"`
	Properties map[string]Property `json:"properties,omitempty"`
}

// Mappings .
type Mappings struct {
	Meta       Meta                `json:"meta,omitempty"` // 元数据，用于字段注释说明
	Properties map[string]Property `json:"properties,omitempty"`
}

// ElasticsearchMapping .
type ElasticsearchMapping struct {
	Mappings Mappings `json:"mappings,omitempty"`
}

// GeneratorOptions defines the optional parameters for the GenerateDatamodel function.
type GeneratorOptions struct {
	InitClassName      *string
	TypeMappingPath    *string
	ExceptionFieldPath *string
	ExceptionTypePath  *string
	SkipFieldPath      *string
	FieldCommentPath   *string
	TmplPath           *string
}

// Field 字段的mapping
type Field struct {
	FieldName     string
	FieldType     string // golang的字段类型
	EsFieldType   string // es的mapping字段类型
	JSONName      string
	FieldComment  string // 字段注释
	FieldsKeyword string // text字段keyword子字段
}

// StructData 结构体数据
type StructData struct {
	PackageName       string
	InitClassName     string
	StructName        string
	StructDefinitions string
}

// GoTypeMap holds the mapping from Elasticsearch types to Go types.
var (
	GoTypeMap       map[string]string
	FieldExceptions map[string]string
	TypeExceptions  map[string]string
	SkipFields      map[string]bool
	FieldComments   map[string]string // 字段注释
)

// StructNameTracker to avoid generating duplicate struct names
var StructNameTracker map[string]bool

// GenerateDataModel 生成es数据的model
func GenerateDataModel(inputPath, outputPath, packageName, structName string, opts *GeneratorOptions) error {
	// check for required fields
	if inputPath == "" || outputPath == "" || structName == "" || packageName == "" {
		return fmt.Errorf("inputPath, outputPath, structName, and packageName must be specified")
	}

	// initialize StructNameTracker
	StructNameTracker = make(map[string]bool)

	// load custom type mapping if provided
	if opts != nil && opts.TypeMappingPath != nil && *opts.TypeMappingPath != "" {
		loadTypeMapping(*opts.TypeMappingPath)
	} else {
		// default mapping
		GoTypeMap = map[string]string{
			"integer":   "int64",
			"float":     "float64",
			"boolean":   "bool",
			"text":      "string",
			"keyword":   "string",
			"date":      "time.Time",
			"geo_point": "[]float64",
			"object":    "map[string]any",
			"nested":    "[]any",
		}
	}

	// load field exceptions if provided
	if opts != nil && opts.ExceptionFieldPath != nil && *opts.ExceptionFieldPath != "" {
		loadFieldExceptions(*opts.ExceptionFieldPath)
	} else {
		FieldExceptions = make(map[string]string)
	}

	// load type exceptions if provided
	if opts != nil && opts.ExceptionTypePath != nil && *opts.ExceptionTypePath != "" {
		loadTypeExceptions(*opts.ExceptionTypePath)
	} else {
		TypeExceptions = make(map[string]string)
	}

	// load skip fields if provided
	if opts != nil && opts.SkipFieldPath != nil && *opts.SkipFieldPath != "" {
		loadSkipFields(*opts.SkipFieldPath)
	} else {
		SkipFields = make(map[string]bool)
	}

	// load field comments if provided
	if opts != nil && opts.FieldCommentPath != nil && *opts.FieldCommentPath != "" {
		loadFieldComments(*opts.FieldCommentPath)
	} else {
		FieldComments = make(map[string]string)
	}

	// load custom template if provided
	var tmpl *template.Template
	var err error
	if opts != nil && opts.TmplPath != nil && *opts.TmplPath != "" {
		tmpl, err = template.ParseFiles(*opts.TmplPath)
		if err != nil {
			return fmt.Errorf("Failed to load template file %s: %v", *opts.TmplPath, err)
		}
	} else {
		// choose default template based on the presence of InitClassName
		if opts != nil && opts.InitClassName != nil && *opts.InitClassName != "" {
			tmpl, err = template.New("structWithWrapper").Parse(structTemplateWithWrapper)
		} else {
			tmpl, err = template.New("structWithoutWrapper").Parse(structTemplateWithoutWrapper)
		}
		if err != nil {
			return fmt.Errorf("Error parsing template: %v", err)
		}
	}

	return processFile(inputPath, outputPath, packageName, structName, opts, tmpl)
}

func processFile(inputPath, outputPath, packageName, structName string, opts *GeneratorOptions, tmpl *template.Template) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("Failed to read file %s: %v", inputPath, err)
	}

	var esMapping ElasticsearchMapping
	err = json.Unmarshal(data, &esMapping)
	if err != nil {
		return fmt.Errorf("Error unmarshalling JSON from file %s: %v", inputPath, err)
	}

	structDefinitions := generateStructDefinitions(structName, esMapping.Mappings.Meta, esMapping.Mappings.Properties)

	var initClassName string
	if opts != nil && opts.InitClassName != nil {
		initClassName = *opts.InitClassName
	}

	structData := StructData{
		PackageName:       packageName,
		InitClassName:     initClassName,
		StructName:        structName,
		StructDefinitions: structDefinitions,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, structData)
	if err != nil {
		return fmt.Errorf("Error executing template: %v", err)
	}

	err = os.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write output file %s: %v", outputPath, err)
	}

	fmt.Printf("Generated Go struct for %s and saved to %s\n", inputPath, outputPath)
	return nil
}

func generateStructDefinitions(structName string, meta Meta, properties map[string]Property) string {
	var structDefs strings.Builder
	generateStruct(&structDefs, structName, meta, properties)
	return structDefs.String()
}

func generateStruct(structDefs *strings.Builder, structName string, meta Meta, properties map[string]Property) {
	// check if the struct has already been generated
	if _, exists := StructNameTracker[structName]; exists {
		return
	}

	// mark this struct as generated
	StructNameTracker[structName] = true

	fields := []Field{}
	nestedStructs := []string{}

	for name, prop := range properties {
		// skip fields that are in the SkipFields map
		if _, skip := SkipFields[name]; skip {
			continue
		}

		fieldName := mapElasticsearchFieldToGoField(name)
		var fieldType string
		var fieldsKeyword string
		fieldComment := prop.Meta.Comment

		if prop.Type == "object" || prop.Type == "nested" {
			// check if the type has a custom exception
			if customType, exists := TypeExceptions[name]; exists {
				var nestedStructName string
				fieldType = customType
				if strings.HasPrefix(fieldType, "*") {
					nestedStructName = fieldType[1:] + "Nested" // "*" 删除
				} else if strings.HasPrefix(fieldType, "[]") {
					nestedStructName = fieldType[2:] + "Nested" // "[]" 删除
				} else {
					nestedStructName = fieldType + "Nested"
				}
				nestedStructs = append(nestedStructs, generateStructDefinitions(nestedStructName, prop.Meta, prop.Properties))
			} else {
				nestedStructName := ToPascalCase(name) + "Nested"
				fieldType = "*" + nestedStructName
				fmt.Println(fieldType)
				nestedStructs = append(nestedStructs, generateStructDefinitions(nestedStructName, prop.Meta, prop.Properties))
			}
		} else {
			fieldType = mapElasticsearchTypeToGoType(name, prop.Type)
			fieldsKeyword = prop.Fields.Keyword.Type
		}

		// 以配置文件为准
		comment := mapElasticsearchFieldToComment(name)
		if comment != "" {
			fieldComment = comment
		}

		fields = append(fields, Field{
			FieldName:     fieldName,
			FieldType:     fieldType,
			EsFieldType:   prop.Type,
			JSONName:      name,
			FieldComment:  fieldComment,
			FieldsKeyword: fieldsKeyword,
		})
	}

	// sort fields alphabetically
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].FieldName < fields[j].FieldName
	})

	// generate struct definition
	if meta.Comment == "" {
		meta.Comment = "."
	}
	structDefs.WriteString(fmt.Sprintf("// %s %s\n", structName, meta.Comment))
	structDefs.WriteString(fmt.Sprintf("type %s struct {\n", structName))
	for _, field := range fields {
		if field.FieldComment != "" {
			if field.FieldsKeyword != "" {
				structDefs.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\" es:\"type:%s;%s\"` // %s\n",
					field.FieldName, field.FieldType, field.JSONName, field.EsFieldType, field.FieldsKeyword, field.FieldComment))
			} else {
				structDefs.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\" es:\"type:%s\"` // %s\n",
					field.FieldName, field.FieldType, field.JSONName, field.EsFieldType, field.FieldComment))

			}
		} else {
			if field.FieldsKeyword != "" {
				structDefs.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\" es:\"type:%s;%s\"`\n",
					field.FieldName, field.FieldType, field.JSONName, field.EsFieldType, field.FieldsKeyword))
			} else {
				structDefs.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\" es:\"type:%s\"`\n",
					field.FieldName, field.FieldType, field.JSONName, field.EsFieldType))
			}
		}
	}
	structDefs.WriteString("}\n\n")

	// append nested structs
	for _, nestedStruct := range nestedStructs {
		structDefs.WriteString(nestedStruct)
	}
}

func mapElasticsearchTypeToGoType(name, esType string) string {
	// check if the type has a custom exception
	if customType, exists := TypeExceptions[name]; exists {
		return customType
	}

	goType, exists := GoTypeMap[esType]
	if !exists {
		goType = "any"
	}

	return goType
}

func mapElasticsearchFieldToGoField(esFieldName string) string {
	// check if the field has a custom exception
	if customFieldName, exists := FieldExceptions[esFieldName]; exists {
		return customFieldName
	}

	return ToPascalCase(esFieldName)
}

func mapElasticsearchFieldToComment(esFieldName string) string {
	// check if the field has a custom comment
	if comment, exists := FieldComments[esFieldName]; exists {
		return comment
	}

	return ""
}

func loadTypeMapping(filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read type mapping file %s: %v", filePath, err)
	}

	err = json.Unmarshal(data, &GoTypeMap)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON from type mapping file %s: %v", filePath, err)
	}
}

func loadFieldExceptions(filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read field exception file %s: %v", filePath, err)
	}

	err = json.Unmarshal(data, &FieldExceptions)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON from field exception file %s: %v", filePath, err)
	}
}

func loadTypeExceptions(filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read type exception file %s: %v", filePath, err)
	}

	err = json.Unmarshal(data, &TypeExceptions)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON from type exception file %s: %v", filePath, err)
	}
}

func loadSkipFields(filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read skip fields file %s: %v", filePath, err)
	}

	err = json.Unmarshal(data, &SkipFields)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON from skip fields file %s: %v", filePath, err)
	}
}

func loadFieldComments(filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read field comments file %s: %v", filePath, err)
	}

	err = json.Unmarshal(data, &FieldComments)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON from field comments file %s: %v", filePath, err)
	}
}

// ToCamelCase 转驼峰模式
func ToCamelCase(s string) string {
	caser := cases.Title(language.Und) // or: `language.English`
	parts := strings.Split(s, "_")
	for i, part := range parts {
		parts[i] = caser.String(part)
	}
	parts[0] = strings.ToLower(parts[0])
	return strings.Join(parts, "")
}

// ToPascalCase .
func ToPascalCase(s string) string {
	caser := cases.Title(language.Und)
	parts := strings.Split(s, "_")
	for i, part := range parts {
		parts[i] = caser.String(part)
	}
	return strings.Join(parts, "")
}

// GeoPoint Elasticsearch的地理坐标
type GeoPoint struct {
	Lat float64 `json:"lat"` // 纬度
	Lon float64 `json:"lon"` // 经度
}
