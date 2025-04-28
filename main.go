package main

import (
	"flag"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"sync"
	"time"

	gen "github.com/kyle-hy/es2go/generator"
)

// GetOrDefault .
func GetOrDefault[T any](ptr *T, defaultVal T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

// FuncName 获取函数名称
func FuncName(f any) string {
	// 获取函数的反射值
	functionValue := reflect.ValueOf(f)

	// 检查是否为函数类型
	if functionValue.Kind() == reflect.Func {
		// 获取函数的名称
		functionName := runtime.FuncForPC(functionValue.Pointer()).Name()
		return functionName
	}
	return ""
}

// GenFunc 代码生成函数定义
type GenFunc func(input, output string, esInfo *gen.EsModelInfo) error

func main() {
	// required arguments
	inputPath := flag.String("in", "", "Input JSON schema mapping file (including file name)")
	outputPath := flag.String("out", "", "Output Go file (including file name)")
	packageName := flag.String("package", "model", "Name of the Go package")
	structName := flag.String("struct", "GeneratedStruct", "Name of the generated Go struct")

	// optional arguments
	initClassName := flag.String("init", "", "Name of the initial wrapper struct (optional)")
	typeMappingPath := flag.String("type-mapping", "", "Path to JSON file specifying Elasticsearch to Go type mapping")
	exceptionFieldPath := flag.String("exception-field", "", "Path to JSON file specifying exceptions for field names")
	exceptionTypePath := flag.String("exception-type", "", "Path to JSON file specifying exceptions for field types")
	skipFieldPath := flag.String("skip-field", "", "Path to JSON file specifying fields to skip")
	fieldCommentPath := flag.String("field-comment", "", "Path to JSON file specifying comments for fields")
	tmplPath := flag.String("tmpl", "", "Path to custom Go template file")

	flag.Parse()

	// validate required arguments
	if *inputPath == "" || *outputPath == "" || *structName == "" || *packageName == "" {
		log.Fatalf("All --in, --out, --struct, and --package must be specified")
	}

	// set up generator options
	opts := &gen.GenOptions{
		InitClassName:      nullableString(initClassName),
		TypeMappingPath:    nullableString(typeMappingPath),
		ExceptionFieldPath: nullableString(exceptionFieldPath),
		ExceptionTypePath:  nullableString(exceptionTypePath),
		SkipFieldPath:      nullableString(skipFieldPath),
		FieldCommentPath:   nullableString(fieldCommentPath),
		TmplPath:           nullableString(tmplPath),
	}

	// 生成struct结构体定义
	esInfo, err := gen.GenEsModel(*inputPath, *outputPath, *packageName, *structName, opts)
	if err != nil {
		log.Fatalf("Failed to generate data model: %v", err)
	}

	tn := time.Now()
	genFuncs := []GenFunc{
		// 生成详情查找函数接口
		gen.GenEsDetailMatch,
		gen.GenEsDetailRange,
		gen.GenEsDetailRecent,
		gen.GenEsDetailVector,

		// 生成聚合分析代码
		gen.GenEsAggMatchTerms,
		gen.GenEsAggMatchStats,
		gen.GenEsAggRangeTerms,
		gen.GenEsAggRangeStats,
		gen.GenEsAggRecentTerms,
		gen.GenEsAggRecentStats,
	}

	// 并发执行各种代码生成函数
	var wg sync.WaitGroup
	wrap := func(f GenFunc, inputPath, outputPath string, esInfo *gen.EsModelInfo) {
		defer wg.Done()
		tn := time.Now()
		f(inputPath, outputPath, esInfo)
		fmt.Printf("%s:  %v\n", FuncName(f), time.Now().Sub(tn))
	}
	for _, gf := range genFuncs {
		wg.Add(1)
		go wrap(gf, *inputPath, *outputPath, esInfo)
	}
	wg.Wait()

	te := time.Now()
	fmt.Println("total: ", te.Sub(tn))
}

// nullableString is a helper function to treat flag.String values as nullable.
func nullableString(flagValue *string) *string {
	if *flagValue == "" {
		return nil
	}
	return flagValue
}
