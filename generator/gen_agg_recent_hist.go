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

// 生成对时间字段近期查找的代码,数值字段范围查找

// PreAggRecentHistCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreAggRecentHistCond(mappingPath string, esInfo *EsModelInfo, rtype string) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 尝试加载自定义生成配置
	genCfg := LoadCustomGenConfig(mappingPath)

	// 根据配置处理全文本字段的配置
	fields := RetainTextFieldByName(esInfo.Fields, genCfg.AllTextFieldOnly, genCfg.AllTextField)

	// 根据配置文件自定义字段分组进行随机组合
	cmbFields := CombineCustom(fields, genCfg.Combine, genCfg.MaxCombine-1)

	// 过滤出满足类型限制的组合
	cmbLimit := map[string]int{TypeNumber: 2, TypeDate: 1, TypeVector: -1}
	limitCmbs := LimitCombineFilter(cmbFields, cmbLimit)

	// 过滤出最少包含指定类型之一的组合
	rangeTypes := []string{TypeNumber, TypeDate}
	mustFileds := MustCombineFilter(limitCmbs, []string{TypeDate})

	// 字段随机组合
	for _, cfs := range mustFileds {
		// 筛选出做聚合分析的类型的字段
		termsFields := FilterOutByTypes(fields, cfs, []string{TypeNumber}, nil)

		// 过滤出配置文件指定的聚合字段
		termsFields = FilterOutByName(termsFields, cfs, genCfg.HistFields, genCfg.NotHistFields)

		// histogram的聚合分析
		termsCmbs := utils.Combinations(termsFields, 1)
		for _, tcmb := range termsCmbs {
			names := getAggRecentHistFuncName(esInfo.StructName, cfs, tcmb, rangeTypes, rtype, genCfg.CmpOptList)
			comments := getAggRecentHistFuncComment(esInfo.StructComment, cfs, tcmb, rangeTypes, rtype, genCfg.CmpOptList)
			params := getAggRecentHistFuncParams(cfs, rangeTypes, rtype, genCfg.CmpOptList)
			queries := getAggRecentHistQuery(cfs, tcmb, rangeTypes, genCfg.TermInShould, rtype, genCfg.CmpOptList)
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

// getAggRecentHistFuncName 获取函数名称
func getAggRecentHistFuncName(structName string, fields, termsFields []*FieldInfo, rangeTypes []string, rtype string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}
	types, other := FieldFilterByTypes(fields, rangeTypes)
	otherName := ""
	// 串联过滤条件的字段名
	if len(other) > 0 {
		for _, f := range other {
			otherName += f.FieldName
		}
	}

	// 各字段与比较符号列表的串联
	fieldOpts := GenRangeFieldName(types, optList, []string{TypeNumber})
	fieldOpts = append(fieldOpts, GenRangeFieldName(types, [][]string{{GTE}}, []string{TypeDate})...)

	names := []string{}
	fn := "Hist" + GenFieldsName(termsFields) + "Of" + rtype + structName + "By" + otherName
	// 多字段之间的两两组合
	fopts := utils.Cartesian(fieldOpts)
	for _, fopt := range fopts {
		names = append(names, fn+fopt)
	}
	return names
}

// 根据发布日期大于等于和小于等于检索books表并分组统计类别的分布情况

// getAggRecentHistFuncComment 获取函数注释
func getAggRecentHistFuncComment(structComment string, fields, termsFields []*FieldInfo, rangeTypes []string, rtype string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	// 函数注释部分
	types, other := FieldFilterByTypes(fields, rangeTypes)
	otherComment := "根据"
	// 串联过滤条件的字段名
	if len(other) > 0 {
		otherComment += GenFieldsCmt(other, false)
	}

	fieldCmts := GenRangeFuncParamCmt(types, optList, []string{TypeNumber})
	fieldCmts = append(fieldCmts, GenRecentFuncParamCmt(types, rtype, optList, []string{TypeDate})...)
	funcCmts := []string{}
	fopts := utils.Cartesian(fieldCmts)
	for _, fopt := range fopts {
		fopt = strings.TrimSuffix(fopt, "、")
		funcCmts = append(funcCmts, otherComment+fopt+"检索"+structComment+"并按"+GenFieldsCmt(termsFields, true)+"区间分桶统计记录数量的直方图分布\n")
	}

	// 参数注释部分
	filterParam := GenParamCmt(other, false)

	// 范围条件部分
	fieldParamCmts := GenRangeParamCmt(types, optList, []string{TypeNumber})
	fieldParamCmts = append(fieldParamCmts, GenRecentParamCmt(types, rtype, optList, []string{TypeDate})...)
	paramOpts := utils.Cartesian(fieldParamCmts)

	// 函数注释和参数注释合并
	if len(funcCmts) == len(paramOpts) {
		for idx, fc := range funcCmts {
			pcmt := fc + filterParam + paramOpts[idx]
			pcmt += "// histInterval float64 分桶聚合的" + GenFieldsCmt(termsFields, true) + "区间间隔"
			funcCmts[idx] = pcmt
		}
	}

	return funcCmts
}

// getAggRecentHistFuncParams 获取函数参数列表
func getAggRecentHistFuncParams(fields []*FieldInfo, rangeTypes []string, rtype string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}
	types, other := FieldFilterByTypes(fields, rangeTypes)
	// 过滤条件参数
	cfp := GenParam(other, false)

	// 范围条件参数
	params := GenRangeParam(types, optList, []string{TypeNumber})
	params = append(params, GenRecentParam(types, rtype, optList, []string{TypeDate})...)

	funcParams := utils.Cartesian(params)
	for idx, fp := range funcParams {
		funcParams[idx] = simplifyParams(cfp + fp + "histInterval float64")
	}
	return funcParams
}

// getAggRecentHistQuery 获取函数的查找条件
func getAggRecentHistQuery(fields, termsFields []*FieldInfo, rangeTypes []string, termInShould bool, rtype string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	types, other := FieldFilterByTypes(fields, rangeTypes)
	// match部分参数
	mq := GenMatchCond(other)

	// agg部分参数
	aq := GenAggWithCondOpt(termsFields, AggFuncHist, AggOptInterval)

	// term部分参数
	eqt := eqTerms(other)

	// 范围条件部分
	ranges := eqRanges(types, optList, []string{TypeNumber})                // 数值的氛围条件
	ranges = append(ranges, eqRecents(types, rtype, []string{TypeDate})...) // 日期的近期条件
	funcRanges := utils.Cartesian(ranges)
	for idx, fq := range funcRanges {
		tq := "terms := []eq.Map{\n" + eqt + fq + "}\n"
		// bool部分
		bq := GenBoolCond(mq, tq, termInShould)

		// esquery部分
		esq := GenESQueryCond(bq, aq, "", "")

		// 拼接match,term和agg条件
		fq := mq + tq + aq + esq

		funcRanges[idx] = fq
	}

	return funcRanges
}

// GenEsAggRecentHist 生成es检索详情
func GenEsAggRecentHist(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	for _, rtype := range RecentTypes {
		// 预处理渲染所需的内容
		funcData := PreAggRecentHistCond(mappingPath, esInfo, rtype)
		detailData := DetailTplData{
			GenFileName:   CurrentFileName(),
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
		suffix := fmt.Sprintf("_agg_%s_hist.go", strings.ToLower(rtype))
		output := strings.Replace(outputPath, ".go", suffix, -1)
		err = os.WriteFile(output, buf.Bytes(), 0644)
		if err != nil {
			return fmt.Errorf("Failed to write output file %s: %v", output, err)
		}

		// 调用go格式化工具格式化代码
		cmd := exec.Command("goimports", "-w", output)
		cmd.Run()
	}

	return nil
}
