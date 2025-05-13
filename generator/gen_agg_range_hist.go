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

// 生成数值和日期字段范围查找的代码

// PreAggRangeHistCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreAggRangeHistCond(mappingPath string, esInfo *EsModelInfo) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 尝试加载自定义生成配置
	genCfg := LoadCustomGenConfig(mappingPath)

	// 根据配置处理全文本字段的配置
	fields := RetainTextFieldByName(esInfo.Fields, genCfg.AllTextFieldOnly, genCfg.AllTextField)

	// 根据配置文件自定义字段分组进行随机组合
	cmbFields := CombineCustom(fields, genCfg.Combine, genCfg.MaxCombine-1)

	// 过滤出满足类型限制的组合
	cmbLimit := map[string]int{TypeNumber: 1, TypeDate: 1, TypeVector: -1}
	limitCmbs := LimitCombineFilter(cmbFields, cmbLimit)

	// 过滤出最少包含指定类型之一的组合
	rangeTypes := []string{TypeNumber, TypeDate}
	mustFileds := MustCombineFilter(limitCmbs, rangeTypes)

	// 字段随机组合
	for _, cfs := range mustFileds {
		// 筛选出做聚合分析的类型的字段
		termsFields := FilterOutByTypes(fields, cfs, []string{TypeNumber}, nil)

		// 过滤出配置文件指定的聚合字段
		termsFields = FilterOutByName(termsFields, cfs, genCfg.HistFields, genCfg.NotHistFields)

		// histogram的聚合分析
		statsCmbs := utils.Combinations(termsFields, 1)
		for _, scmb := range statsCmbs {
			names := getAggRangeHistFuncName(esInfo.StructName, cfs, scmb, rangeTypes, genCfg.CmpOptList)
			comments := getAggRangeHistFuncComment(esInfo.StructComment, cfs, scmb, rangeTypes, genCfg.CmpOptList)
			params := getAggRangeHistFuncParams(cfs, rangeTypes, genCfg.CmpOptList)
			queries := getAggRangeHistQuery(cfs, scmb, rangeTypes, genCfg.TermInShould, genCfg.CmpOptList)
			for idx := range len(names) {
				ftd := &FuncTplData{
					Name:    names[idx],
					Comment: comments[idx],
					Params:  params[idx],
					Query:   queries[idx],
				}
				funcDatas = append(funcDatas, ftd)
			}
		}
	}

	return funcDatas
}

// getAggRangeHistFuncName 获取函数名称
func getAggRangeHistFuncName(structName string, fields, termsFields []*FieldInfo, rangeTypes []string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	types, other := FieldFilterByTypes(fields, rangeTypes)
	otherName := ""
	// 串联过滤条件的字段名
	if len(other) > 0 {
		otherName = GenFieldsName(other)
	}

	// 各字段与比较符号列表的串联
	fieldOpts := GenRangeFieldName(types, optList, nil)

	names := []string{}
	fn := "Hist" + GenFieldsName(termsFields) + "Of" + structName + "By"
	// 多字段之间的两两组合
	fopts := utils.Cartesian(fieldOpts)
	for _, fopt := range fopts {
		names = append(names, fn+fopt+otherName)
	}
	return names
}

// getAggRangeHistFuncComment 获取函数注释
func getAggRangeHistFuncComment(structComment string, fields, termsFields []*FieldInfo, rangeTypes []string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}
	statCmt := "并统计"

	// 函数注释部分
	types, other := FieldFilterByTypes(fields, rangeTypes)
	otherComment := "根据"
	// 串联过滤条件的字段名
	if len(other) > 0 {
		otherComment += GenFieldsCmt(other, false)
	}

	// 范围查询字段比较方式
	fieldCmts := GenRangeFuncParamCmt(types, optList, nil)

	funcCmts := []string{}
	fopts := utils.Cartesian(fieldCmts)
	for _, fopt := range fopts {
		fopt = strings.TrimSuffix(fopt, "、")
		fcmt := otherComment + fopt + "检索" + structComment + statCmt + GenFieldsCmt(termsFields, true) + "的数量直方图分布\n"
		funcCmts = append(funcCmts, fcmt)
	}

	// 参数注释部分
	// 过滤条件部分
	filterParam := GenParamCmt(other, false)

	// 范围条件部分
	fieldParamCmts := GenRangeParamCmt(types, optList, nil)

	// 函数注释和参数注释合并
	paramOpts := utils.Cartesian(fieldParamCmts)
	if len(funcCmts) == len(paramOpts) {
		for idx, fc := range funcCmts {
			pcmt := fc + filterParam + paramOpts[idx]
			pcmt += "// histInterval float64 桶聚合的" + GenFieldsCmt(termsFields, true) + "间隔"
			funcCmts[idx] = pcmt
		}
	}

	return funcCmts
}

// getAggRangeHistFuncParams 获取函数参数列表
func getAggRangeHistFuncParams(fields []*FieldInfo, rangeTypes []string, optList [][]string) []string {
	types, other := FieldFilterByTypes(fields, rangeTypes)
	// 过滤条件参数
	cfp := GenParam(other, false)

	// 范围条件参数
	params := GenRangeParam(types, optList, nil)

	funcParams := utils.Cartesian(params)
	for idx, fp := range funcParams {
		funcParams[idx] = simplifyParams(cfp + fp + "histInterval float64")
	}
	return funcParams
}

// getAggRangeHistQuery 获取函数的查找条件
func getAggRangeHistQuery(fields, termsFields []*FieldInfo, rangeTypes []string, termInShould bool, optList [][]string) []string {
	types, other := FieldFilterByTypes(fields, rangeTypes)
	// match部分参数
	mq := GenMatchCond(other)

	// agg部分参数
	aq := GenAggWithCondOpt(termsFields, AggFuncHist, AggOptInterval)

	// term部分和range部分组合
	termRanges := GenTermRangeCond(other, types, optList)
	for idx, trq := range termRanges {
		// bool部分
		bq := GenBoolCond(mq, trq, termInShould)

		// esquery部分
		esq := GenESQueryCond(bq, aq, "", "")

		// 拼接match,term和agg条件
		fq := mq + trq + aq + esq

		termRanges[idx] = fq
	}

	return termRanges
}

// GenEsAggRangeHist 生成es检索详情
func GenEsAggRangeHist(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	// 预处理渲染所需的内容
	funcData := PreAggRangeHistCond(mappingPath, esInfo)
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
	outputPath = strings.Replace(outputPath, ".go", "_agg_range_hist.go", -1)
	err = os.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write output file %s: %v", outputPath, err)
	}

	// 调用go格式化工具格式化代码
	cmd := exec.Command("goimports", "-w", outputPath)
	cmd.Run()

	return nil
}
