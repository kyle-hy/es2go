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

// 生成对text字段检索的代码

// PreDetailTermCond 使用go代码预处理渲染需要的一些逻辑，template脚本出来调试困难
func PreDetailTermCond(esInfo *EsModelInfo) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 按数据类型分组字段
	grpFileds := GroupFieldsByType(esInfo.Fields)

	// 提取目标字段
	fields := grpFileds[TypeKeyword]                  // keyword字段
	fields = append(fields, grpFileds[TypeNumber]...) // 数值

	// 字段随机组合
	cmbFields := utils.Combinations(fields, MaxCombine)
	for _, cfs := range cmbFields {
		ftd := &FuncTplData{
			Name:    getDetailTermFuncName(esInfo.StructName, cfs),
			Comment: getDetailTermFuncComment(esInfo.StructComment, cfs),
			Params:  getDetailTermFuncParams(cfs),
			Query:   getDetailTermMatchQuery(cfs),
		}
		funcDatas = append(funcDatas, ftd)
	}

	return funcDatas
}

// getDetailTermFuncName 获取函数名称
func getDetailTermFuncName(structName string, fields []*FieldInfo) string {
	fn := "Term" + structName + "By"
	for _, f := range fields {
		fn += f.FieldName
	}
	return fn
}

// getDetailTermFuncComment 获取函数注释
func getDetailTermFuncComment(structComment string, fields []*FieldInfo) string {
	// 函数注释
	cmt := "以"
	for _, f := range fields {
		cmt += f.FieldComment + "、"
	}
	cmt = strings.TrimSuffix(cmt, "、")
	cmt += "为条件精确查询" + structComment + "的详细数据列表和总数量"

	// 参数注释
	for _, f := range fields {
		cmt += "\n// " + utils.ToFirstLower(f.FieldName) + " " + f.FieldType + " " + f.FieldComment
	}

	return cmt
}

// getDetailTermFuncParams 获取函数参数列表
func getDetailTermFuncParams(fields []*FieldInfo) string {
	fp := ""
	for _, f := range fields {
		fp += utils.ToFirstLower(f.FieldName) + " " + f.FieldType + ", "
	}
	fp = strings.TrimSuffix(fp, ", ")
	return fp
}

// getDetailTermMatchQuery 获取函数的查询条件
func getDetailTermMatchQuery(fields []*FieldInfo) string {
	fq := ""
	if len(fields) == 1 {
		f := fields[0]
		fq = "esQuery := &eq.ESQuery{\n"
		fq += fmt.Sprintf("		Query: eq.Term(\"%s\", %s),\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName))
		fq += "	}\n"
	} else {
		fq = "terms := []eq.Map{\n"
		for _, f := range fields {
			fq += fmt.Sprintf("		eq.Term(\"%s\", %s),\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName))
		}
		fq += "	}\n"

		fq += `	esQuery := &eq.ESQuery{Query: eq.Bool(eq.WithFilter(terms))}`
	}
	return fq
}

// GenEsDetailTerm 生成es检索详情
func GenEsDetailTerm(outputPath string, esInfo *EsModelInfo) error {
	// 预处理渲染所需的内容
	funcData := PreDetailTermCond(esInfo)
	detailData := DetailTplData{
		PackageName:   esInfo.PackageName,
		StructName:    esInfo.StructName,
		StructComment: esInfo.StructComment,
		IndexName:     esInfo.IndexName,
		FuncDatas:     funcData,
	}

	// 渲染
	tmpl, err := template.New("structDatail").Parse(DetailTpl)
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, detailData)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 写入文件
	outputPath = strings.Replace(outputPath, ".go", "_detail_term.go", -1)
	err = os.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write output file %s: %v", outputPath, err)
	}

	// 调用go格式化工具格式化代码
	cmd := exec.Command("goimports", "-w", outputPath)
	cmd.Run()

	return nil
}
