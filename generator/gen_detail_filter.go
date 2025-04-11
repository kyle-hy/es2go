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

// 生成根据keyword字段作为filter对text字段检索的代码

// PreDetailFilterCond 使用go代码预处理渲染需要的一些逻辑，template脚本出来调试困难
func PreDetailFilterCond(esInfo *EsModelInfo) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 按数据类型分组字段
	grpFileds := GroupFieldsByType(esInfo.Fields)

	// 提取所需字段
	textFields := grpFileds[TypeText]       // 文本字段
	keywordFields := grpFileds[TypeKeyword] // keyword字段

	// 随机组合条件
	cmbTextFields := utils.Combinations(textFields, 1)                  // 组合text字段
	cmbFeywordFields := utils.Combinations(keywordFields, MaxCombine-1) // 随机组合keyword过滤条件
	utils.JPrint(grpFileds)
	cmbFields := utils.CombineSlices(cmbFeywordFields, cmbTextFields) // filter和match条件组合

	for _, cfs := range cmbFields {
		ftd := &FuncTplData{
			Name:    getDetailFilterFuncName(esInfo.StructName, cfs),
			Comment: getDetailFilterFuncComment(esInfo.StructComment, cfs),
			Params:  getDetailFilterFuncParams(cfs),
			Query:   getDetailFilterMatchQuery(cfs),
		}
		funcDatas = append(funcDatas, ftd)
	}

	utils.JPrint(funcDatas)
	return funcDatas
}

// getDetailFilterFuncName 获取函数名称
func getDetailFilterFuncName(structName string, fields [][]*FieldInfo) string {
	filterFields := fields[0]
	testFields := fields[1]

	fn := "Query" + structName + "By"
	for _, f := range testFields {
		fn += f.FieldName
	}

	fn += "Filter"
	for _, f := range filterFields {
		fn += f.FieldName
	}

	return fn
}

// getDetailFilterFuncComment 获取函数注释
func getDetailFilterFuncComment(structComment string, fields [][]*FieldInfo) string {
	filterFields := fields[0]
	testFields := fields[1]

	// 函数注释
	cmt := "以"
	for _, f := range filterFields {
		cmt += f.FieldComment + "、"
	}
	cmt = strings.TrimSuffix(cmt, "、")
	cmt += "为过滤条件对"
	for _, f := range testFields {
		cmt += f.FieldComment + "、"
	}
	cmt = strings.TrimSuffix(cmt, "、")
	cmt += "进行检索查询" + structComment + "的详细数据"

	// 参数注释
	for _, f := range filterFields {
		cmt += "\n// " + utils.ToFirstLower(f.FieldName) + " " + f.FieldType + " " + f.FieldComment
	}
	for _, f := range testFields {
		cmt += "\n// " + utils.ToFirstLower(f.FieldName) + " " + f.FieldType + " " + f.FieldComment
	}

	return cmt
}

// getDetailFilterFuncParams 获取函数参数列表
func getDetailFilterFuncParams(fields [][]*FieldInfo) string {
	filterFields := fields[0]
	testFields := fields[1]

	fp := ""
	for _, f := range filterFields {
		fp += utils.ToFirstLower(f.FieldName) + " " + f.FieldType + ", "
	}

	for _, f := range testFields {
		fp += utils.ToFirstLower(f.FieldName) + " " + f.FieldType + ", "
	}
	fp = strings.TrimSuffix(fp, ", ")

	return fp
}

// getDetailFilterMatchQuery 获取函数的查询条件
func getDetailFilterMatchQuery(fields [][]*FieldInfo) string {
	filterFields := fields[0]
	testFields := fields[1]

	// filter条件
	fq := "filters:= []eq.Map{\n"
	for _, f := range filterFields {
		fq += fmt.Sprintf("		eq.Term(\"%s\", %s),\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName))
	}
	fq += "	}\n"

	// match条件
	fq += "	matches := []eq.Map{\n"
	for _, f := range testFields {
		fq += fmt.Sprintf("		eq.Match(\"%s\", %s),\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName))
	}
	fq += "	}\n"

	fq += `	esQuery := &eq.ESQuery{Query: eq.Bool(eq.WithMust(filters), eq.WithMust(matches))}`
	return fq
}

// GenEsDetailFilter 生成es检索详情
func GenEsDetailFilter(outputPath string, esInfo *EsModelInfo) error {
	// 预处理渲染所需的内容
	funcData := PreDetailFilterCond(esInfo)

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
	outputPath = strings.Replace(outputPath, ".go", "_datail_filter.go", -1)
	err = os.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write output file %s: %v", outputPath, err)
	}

	// 调用go格式化工具格式化代码
	cmd := exec.Command("goimports", "-w", outputPath)
	cmd.Run()

	return nil
}
