package generator

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/kyle-hy/es2go/utils"
)

// 生成对vector字段向量匹配的代码

// PreDetailVectorCond 使用go代码预处理渲染需要的一些逻辑，template脚本出来调试困难
func PreDetailVectorCond(mappingPath string, esInfo *EsModelInfo) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 尝试加载自定义生成配置
	genCfg := LoadCustomGenConfig(mappingPath)
	maxCombine := MaxCombine
	if genCfg.MaxCombine > 0 {
		maxCombine = genCfg.MaxCombine
	}

	// 根据配置处理全文本字段的配置
	fields := esInfo.Fields
	if genCfg.AllTextFieldOnly && genCfg.AllTextField != "" {
		fields = DropTextFieldByName(esInfo.Fields, genCfg.AllTextField)
	}

	// 根据配置文件自定义字段分组进行随机组合
	cmbFields := CombineCustom(fields, genCfg.Combine, maxCombine)
	if len(cmbFields) == 0 { // 不存在自定义字段的配置，则全字段随机
		cmbFields = utils.Combinations(fields, maxCombine)
	}

	// 过滤出满足类型限制的组合
	cmbLimit := map[string]int{TypeVector: 1}
	cmbFields = LimitCombineFilter(cmbFields, cmbLimit)

	knnTypes := []string{TypeVector}
	cmbFields = MustCombineFilter(cmbFields, knnTypes)

	// 构造渲染模板所需的数据
	for _, cfs := range cmbFields {
		ftd := &FuncTplData{
			Name:    getDetailVectorFuncName(esInfo.StructName, cfs, knnTypes),
			Comment: getDetailVectorFuncComment(esInfo.StructComment, cfs, knnTypes),
			Params:  getDetailVectorFuncParams(cfs, knnTypes),
			Query:   getDetailVectorQuery(cfs, genCfg.TermInShould, knnTypes),
		}
		funcDatas = append(funcDatas, ftd)
	}

	return funcDatas
}

// getDetailVectorFuncName 获取函数名称
func getDetailVectorFuncName(structName string, fields []*FieldInfo, rangeTypes []string) string {
	types, other := FieldFilterByTypes(fields, rangeTypes)
	otherName := ""
	// 串联过滤条件的字段名
	if len(other) > 0 {
		otherName = "With" + GenFieldsName(other)
	}

	// 各字段与比较符号列表的串联
	knnName := "By" + GenFieldsName(types)

	fn := "Knn" + structName + knnName + otherName

	return fn
}

// getDetailVectorFuncComment 获取函数注释
func getDetailVectorFuncComment(structComment string, fields []*FieldInfo, rangeTypes []string) string {
	// 函数注释部分
	funcCmt := "对"
	types, other := FieldFilterByTypes(fields, rangeTypes)
	if len(other) > 0 {
		otherCmt := "根据" + GenFieldsCmt(other, true)
		otherCmt += "过滤后"
		funcCmt = otherCmt + funcCmt
	}

	funcCmt += GenFieldsCmt(types, true)
	funcCmt += "进行检索查找" + structComment + "的详细数据列表和总数量\n"

	// 参数注释部分
	paramCmt := GenParamCmt(other, false)
	// 范围条件部分
	paramCmt += GenParamCmt(types, true)
	funcCmt += paramCmt
	return funcCmt
}

// getDetailVectorFuncParams 获取函数参数列表
func getDetailVectorFuncParams(fields []*FieldInfo, rangeTypes []string) string {
	types, other := FieldFilterByTypes(fields, rangeTypes)
	fp := GenParam(other, false)
	fp += GenParam(types, true)
	return fp
}

// getDetailVectorQuery 获取函数的查找条件
func getDetailVectorQuery(fields []*FieldInfo, termInShould bool, rangeTypes []string) string {
	types, other := FieldFilterByTypes(fields, rangeTypes)
	mq := GenFilterCond(other) // 过滤条件
	bq := GenBoolCond(mq, "", termInShould)
	kq := GenKnnCond(types, bq)
	esq := GenESQueryCond("knn", "", "", "")
	return mq + kq + esq
}

// GenEsDetailVector 生成es检索详情
func GenEsDetailVector(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	// 预处理渲染所需的内容
	funcData := PreDetailVectorCond(mappingPath, esInfo)
	detailData := DetailTplData{
		PackageName:   esInfo.PackageName,
		StructName:    esInfo.StructName,
		StructComment: esInfo.StructComment,
		IndexName:     esInfo.IndexName,
		FuncDatas:     funcData,
	}

	// 创建 FuncMap，将函数名映射到 Go 函数
	funcMap := template.FuncMap{
		"FirstLine": utils.FirstLine,
	}

	// 渲染
	tmpl, err := template.New("structDatail").Funcs(funcMap).Parse(DetailTpl)
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, detailData)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 写入文件
	outputPath = strings.Replace(outputPath, ".go", "_detail_vector.go", -1)
	err = os.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write output file %s: %v", outputPath, err)
	}

	// 调用go格式化工具格式化代码
	cmd := exec.Command("goimports", "-w", outputPath)
	cmd.Run()

	return nil
}
