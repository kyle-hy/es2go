package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/kyle-hy/es2go/utils"
)

// GenOptions defines the optional parameters for the GenerateDatamodel function.
type GenOptions struct {
	InitClassName      *string
	TypeMappingPath    *string
	ExceptionFieldPath *string
	ExceptionTypePath  *string
	SkipFieldPath      *string
	FieldCommentPath   *string
	TmplPath           *string
}

// GoTypeMap holds the mapping from Elasticsearch types to Go types.
var (
	GoTypeMap       map[string]string // es字段与go字段的映射
	FieldExceptions map[string]string // 异常字段
	TypeExceptions  map[string]string // 异常类型
	SkipFields      map[string]bool   // 忽略的字段
	FieldComments   map[string]string // 字段注释
)

// StructNameTracker 用于避免生成重复的结构体名称
var StructNameTracker map[string]bool

// GenEsModel 生成es表属性的model
func GenEsModel(inputPath, outputPath, packageName, structName string, opts *GenOptions) (*EsModelInfo, error) {
	// check for required fields
	if inputPath == "" || outputPath == "" || structName == "" || packageName == "" {
		return nil, fmt.Errorf("inputPath, outputPath, structName, and packageName must be specified")
	}

	// initialize StructNameTracker
	StructNameTracker = make(map[string]bool)

	// load custom type mapping if provided
	if opts != nil && opts.TypeMappingPath != nil && *opts.TypeMappingPath != "" {
		loadTypeMapping(*opts.TypeMappingPath)
	} else {
		// default mapping
		GoTypeMap = map[string]string{
			"integer":       "int64",
			"long":          "int64",
			"float":         "float64",
			"boolean":       "bool",
			"text":          "string",
			"keyword":       "string",
			"date":          "time.Time",
			"geo_point":     "[]float64",
			"object":        "map[string]any",
			"nested":        "[]any",
			"dense_vector":  "[]float32",
			"sparse_vector": "[]float32",
			"vector":        "[]float32",
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
			return nil, fmt.Errorf("Failed to load template file %s: %v", *opts.TmplPath, err)
		}
	} else {
		// choose default template based on the presence of InitClassName
		if opts != nil && opts.InitClassName != nil && *opts.InitClassName != "" {
			tmpl, err = template.New("structWithWrapper").Parse(StructTplWithWrapper)
		} else {
			tmpl, err = template.New("structWithoutWrapper").Parse(StructTplWithoutWrapper)
		}
		if err != nil {
			return nil, fmt.Errorf("Error parsing template: %v", err)
		}
	}

	return processFile(inputPath, outputPath, packageName, structName, opts, tmpl)
}

// RemoveExt 删除文件的后缀名
func RemoveExt(path string) string {
	ext := filepath.Ext(path)
	return path[:len(path)-len(ext)]
}

func processFile(inputPath, outputPath, packageName, structName string, opts *GenOptions, tmpl *template.Template) (*EsModelInfo, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %s: %v", inputPath, err)
	}

	var esMapping ElasticsearchMapping
	err = json.Unmarshal(data, &esMapping)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling JSON from file %s: %v", inputPath, err)
	}

	fields, structDefinitions := generateStructDefinitions(structName, esMapping.Mappings.Meta, esMapping.Mappings.Properties, "")

	var initClassName string
	if opts != nil && opts.InitClassName != nil {
		initClassName = *opts.InitClassName
	}

	structData := StructTplData{
		PackageName:       packageName,
		InitClassName:     initClassName,
		StructName:        structName,
		StructDefinitions: structDefinitions,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, structData)
	if err != nil {
		return nil, fmt.Errorf("Error executing template: %v", err)
	}

	err = os.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return nil, fmt.Errorf("Failed to write output file %s: %v", outputPath, err)
	}

	// 调用go格式化工具格式化代码
	cmd := exec.Command("goimports", "-w", outputPath)
	cmd.Run()

	fmt.Printf("Generated Go struct for %s and saved to %s\n", inputPath, outputPath)

	// 从mapping文件名提取es索引名称
	indexName := RemoveExt(filepath.Base(inputPath))
	indexName = strings.TrimSuffix(indexName, "_mapping") // 尝试删除索引文件添加的后缀
	indexName = strings.TrimSuffix(indexName, "-mapping") // 尝试删除索引文件添加的后缀
	esModelInfo := &EsModelInfo{
		PackageName:   packageName,
		InitClassName: initClassName,
		StructName:    structName,
		IndexName:     indexName,
		StructComment: esMapping.Mappings.Meta.Comment,
		Fields:        fields,
	}
	if esModelInfo.StructComment == "" {
		esModelInfo.StructComment = indexName
	}
	return esModelInfo, nil
}

// generateStructDefinitions 生成模型结构体定义的字段信息
func generateStructDefinitions(structName string, meta Meta, properties map[string]Property, nestedPath string) ([]*FieldInfo, string) {
	var structDefs strings.Builder
	fields := generateStruct(&structDefs, structName, meta, properties)
	if nestedPath != "" {
		AddNestedFilePath(nestedPath, fields)
	}
	return fields, structDefs.String()
}

func generateStruct(structDefs *strings.Builder, structName string, meta Meta, properties map[string]Property) []*FieldInfo {
	// check if the struct has already been generated
	if _, exists := StructNameTracker[structName]; exists {
		return nil
	}

	// mark this struct as generated
	StructNameTracker[structName] = true

	fields := []*FieldInfo{}
	allFields := []*FieldInfo{}
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
					nestedStructName = fieldType[1:] // "*" 删除
				} else if strings.HasPrefix(fieldType, "[]") {
					nestedStructName = fieldType[2:] // "[]" 删除
				} else {
					nestedStructName = fieldType
				}

				nestedFields, structDefine := generateStructDefinitions(nestedStructName, prop.Meta, prop.Properties, name)
				nestedStructs = append(nestedStructs, structDefine)

				// AddNestedFilePath(name, nestedFields)
				allFields = append(allFields, nestedFields...)
			} else {
				nestedStructName := utils.ToPascalCase(name)
				fieldType = "*" + nestedStructName

				nestedFields, structDefine := generateStructDefinitions(nestedStructName, prop.Meta, prop.Properties, name)
				nestedStructs = append(nestedStructs, structDefine)

				// AddNestedFilePath(name, nestedFields)
				allFields = append(allFields, nestedFields...)
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

		if fieldComment == "" {
			fieldComment = name
		}

		finfo := &FieldInfo{
			FieldName:     fieldName,
			FieldType:     fieldType,
			EsFieldType:   prop.Type,
			EsFieldPath:   name,
			JSONName:      name,
			FieldComment:  fieldComment,
			FieldsKeyword: fieldsKeyword,
		}
		fields = append(fields, finfo)
		allFields = append(allFields, finfo)
	}

	// sort fields alphabetically
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].FieldName < fields[j].FieldName
	})
	sort.Slice(allFields, func(i, j int) bool {
		return allFields[i].FieldName < allFields[j].FieldName
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

	return allFields
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

// AddNestedFilePath 添加嵌套字段的访问路径
func AddNestedFilePath(nestedName string, fields []*FieldInfo) {
	for _, field := range fields {
		field.EsFieldPath = fmt.Sprintf("%s.%s", nestedName, field.EsFieldPath)
	}
}

func mapElasticsearchFieldToGoField(esFieldName string) string {
	// check if the field has a custom exception
	if customFieldName, exists := FieldExceptions[esFieldName]; exists {
		return customFieldName
	}

	return utils.ToPascalCase(esFieldName)
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

// GeoPoint Elasticsearch的地理坐标
type GeoPoint struct {
	Lat float64 `json:"lat"` // 纬度
	Lon float64 `json:"lon"` // 经度
}

/**************** 渲染相关 *************/

// StructTplData 模板渲染传入的结构体数据
type StructTplData struct {
	PackageName       string // 包名
	InitClassName     string // 封装的类型名称
	StructName        string // 模型结构体名称
	StructDefinitions string // 模型结构体定义，所有属性的渲染都已在go代码实现
}

// StructTplWithWrapper .
const StructTplWithWrapper = `// Code generated by es2go. DO NOT EDIT.
package {{.PackageName}}

import "time"

type {{.InitClassName}} struct {
	{{.StructName}}
}

{{.StructDefinitions}}
`

// StructTplWithoutWrapper .
const StructTplWithoutWrapper = `// Code generated by es2go. DO NOT EDIT.

package {{.PackageName}}

import "time"

{{.StructDefinitions}}
`
