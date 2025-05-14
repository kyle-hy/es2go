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

// PreAggRecentDateHistCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreAggRecentDateHistCond(mappingPath string, esInfo *EsModelInfo, rtype string) []*FuncTplData {
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
	mustFileds := MustCombineFilter(limitCmbs, []string{TypeDate})

	// 字段随机组合
	rangeTypes := []string{TypeNumber, TypeDate}
	for _, cfs := range mustFileds {
		// 筛选出做日期桶聚合的单位维度
		dhtypes := FindPreElem(DateHistTypes, rtype, 2)
		for _, dhtype := range dhtypes {
			names := getAggRecentDateHistFuncName(esInfo.StructName, cfs, rangeTypes, rtype, genCfg.CmpOptList, dhtype)
			comments := getAggRecentDateHistFuncComment(esInfo.StructComment, cfs, rangeTypes, rtype, genCfg.CmpOptList, dhtype)
			params := getAggRecentDateHistFuncParams(cfs, rangeTypes, rtype, genCfg.CmpOptList)
			queries := getAggRecentDateHistQuery(cfs, rangeTypes, genCfg.TermInShould, rtype, genCfg.CmpOptList, dhtype)
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

// getAggRecentDateHistFuncName 获取函数名称
func getAggRecentDateHistFuncName(structName string, fields []*FieldInfo, rangeTypes []string, rtype string, optList [][]string, dhtype string) []string {
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
	fn := dhtype + "HistOf" + rtype + structName + "By" + otherName

	// 多字段之间的两两组合
	fopts := utils.Cartesian(fieldOpts)
	for _, fopt := range fopts {
		names = append(names, fn+fopt)
	}
	return names
}

// 根据发布日期大于等于和小于等于检索books表并分组统计类别的分布情况

// getAggRecentDateHistFuncComment 获取函数注释
func getAggRecentDateHistFuncComment(structComment string, fields []*FieldInfo, rangeTypes []string, rtype string, optList [][]string, dhtype string) []string {
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
		funcCmts = append(funcCmts, otherComment+fopt+"检索"+structComment+"并分桶统计"+DateHistNames[dhtype]+"的记录数量直方图分布\n")
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
			funcCmts[idx] = strings.TrimSuffix(pcmt, "\n")
		}
	}

	return funcCmts
}

// getAggRecentDateHistFuncParams 获取函数参数列表
func getAggRecentDateHistFuncParams(fields []*FieldInfo, rangeTypes []string, rtype string, optList [][]string) []string {
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
		funcParams[idx] = simplifyParams(cfp + fp)
	}
	return funcParams
}

// getAggRecentDateHistQuery 获取函数的查找条件
func getAggRecentDateHistQuery(fields []*FieldInfo, rangeTypes []string, termInShould bool, rtype string, optList [][]string, dhtype string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	types, other := FieldFilterByTypes(fields, rangeTypes)
	// match部分参数
	mq := GenMatchCond(other)

	// agg部分参数
	dateFields, _ := FieldFilterByTypes(fields, []string{TypeDate})
	aq := GenAggWithCondOpt(dateFields, AggFuncDateHist, fmt.Sprintf(AggOptCalendarInterval, DateHistInterval[dhtype]))

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

// FindPreElem 遍历切片寻找目标元素
func FindPreElem(strs []string, elem string, preNum int) []string {
	for i, str := range strs {
		if str == elem {
			// 计算开始索引，防止越界
			start := max(i-preNum, 0)

			// 截取从start到当前元素位置（包括当前元素）的子切片
			end := i + 1 // 当前元素位置的下一个位置，确保当前元素被包含在内
			return strs[start:end]
		}
	}
	// 如果没有找到该元素，则返回空切片
	return []string{}
}

// GenEsAggRecentDateHist 生成es检索详情
func GenEsAggRecentDateHist(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	for _, rtype := range RecentTypes {
		// 预处理渲染所需的内容
		funcData := PreAggRecentDateHistCond(mappingPath, esInfo, rtype)
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
		suffix := fmt.Sprintf("_agg_%s_date_hist.go", strings.ToLower(rtype))
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
