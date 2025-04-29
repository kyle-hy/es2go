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

// PreDetailRecentCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreDetailRecentCond(mappingPath string, esInfo *EsModelInfo, rtype string) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 尝试加载自定义生成配置
	genCfg := LoadCustomGenConfig(mappingPath)
	maxCombine := MaxCombine
	if genCfg.MaxCombine > 0 {
		maxCombine = genCfg.MaxCombine
	}

	// 根据配置处理全文本字段的配置
	fields := RetainTextFieldByName(esInfo.Fields, genCfg.AllTextFieldOnly, genCfg.AllTextField)

	// 根据配置文件自定义字段分组进行随机组合
	cmbFields := CombineCustom(fields, genCfg.Combine, maxCombine)
	if len(cmbFields) == 0 { // 不存在自定义字段的配置，则全字段随机
		cmbFields = utils.Combinations(fields, maxCombine)
	}

	// 过滤出满足类型限制的组合
	cmbLimit := map[string]int{TypeNumber: 2, TypeDate: 1, TypeVector: -1}
	limitCmbs := LimitCombineFilter(cmbFields, cmbLimit)

	// 过滤出最少包含指定类型之一的组合
	rangeTypes := []string{TypeNumber, TypeDate}
	mustFileds := MustCombineFilter(limitCmbs, []string{TypeDate})

	// 字段随机组合
	for _, cfs := range mustFileds {
		names := getDetailRecentFuncName(esInfo.StructName, cfs, rangeTypes, rtype, genCfg.CmpOptList)
		comments := getDetailRecentFuncComment(esInfo.StructComment, cfs, rangeTypes, rtype, genCfg.CmpOptList)
		params := getDetailRecentFuncParams(cfs, rangeTypes, rtype, genCfg.CmpOptList)
		queries := getDetailRecentQuery(cfs, rangeTypes, genCfg.TermInShould, rtype, genCfg.CmpOptList)
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

	return funcDatas
}

// getDetailRecentFuncName 获取函数名称
func getDetailRecentFuncName(structName string, fields []*FieldInfo, rangeTypes []string, rtype string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}
	types, other := FieldFilterByTypes(fields, rangeTypes)
	otherName := ""
	// 串联过滤条件的字段名
	if len(other) > 0 {
		otherName = "With" + GenFieldsName(other)
	}

	fieldOpts := GenRangeFieldName(types, optList, []string{TypeNumber})
	fieldOpts = append(fieldOpts, GenRangeFieldName(types, [][]string{{GTE}}, []string{TypeDate})...)

	names := []string{}
	fn := rtype + structName + "By"
	// 多字段之间的两两组合
	fopts := utils.Cartesian(fieldOpts)
	for _, fopt := range fopts {
		names = append(names, fn+fopt+otherName)
	}
	return names
}

// getDetailRecentFuncComment 获取函数注释
func getDetailRecentFuncComment(structComment string, fields []*FieldInfo, rangeTypes []string, rtype string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}
	// 函数注释部分
	types, other := FieldFilterByTypes(fields, rangeTypes)
	otherComment := ""
	// 串联过滤条件的字段名
	if len(other) > 0 {
		otherComment = "根据" + GenFieldsCmt(other, true)
	}

	fieldCmts := GenRangeFuncParamCmt(types, optList, []string{TypeNumber})
	fieldCmts = append(fieldCmts, GenRecentFuncParamCmt(types, rtype, optList, []string{TypeDate})...)
	funcCmts := []string{}
	fn := "从" + structComment + "查找"
	fopts := utils.Cartesian(fieldCmts)
	for _, fopt := range fopts {
		fopt = strings.TrimSuffix(fopt, "、")
		funcCmts = append(funcCmts, otherComment+fn+fopt+"的详细数据列表和总数量\n")
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
			funcCmts[idx] = fc + filterParam + strings.TrimSuffix(paramOpts[idx], "\n")
		}
	}

	return funcCmts
}

// getDetailRecentFuncParams 获取函数参数列表
func getDetailRecentFuncParams(fields []*FieldInfo, rangeTypes []string, rtype string, optList [][]string) []string {
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
		funcParams[idx] = simplifyParams(cfp + strings.TrimSuffix(fp, ", "))
	}
	return funcParams
}

// getDetailRecentQuery 获取函数的查找条件
func getDetailRecentQuery(fields []*FieldInfo, rangeTypes []string, termInShould bool, rtype string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	types, other := FieldFilterByTypes(fields, rangeTypes)
	// match部分参数
	mq := GenMatchCond(other)

	// term部分参数
	tq := eqTerms(other)

	ranges := eqRanges(types, optList, []string{TypeNumber})                // 数值的氛围条件
	ranges = append(ranges, eqRecents(types, rtype, []string{TypeDate})...) // 日期的近期条件
	funcRanges := utils.Cartesian(ranges)
	for idx, fq := range funcRanges {
		fq = WrapTermCond(tq + fq)
		bq := GenBoolCond(mq, fq, termInShould)
		esq := GenESQueryCond(bq, "")
		funcRanges[idx] = mq + fq + esq
	}

	return funcRanges
}

// GenEsDetailRecent 生成es检索详情
func GenEsDetailRecent(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	for _, rtype := range RecentTypes {
		// 预处理渲染所需的内容
		funcData := PreDetailRecentCond(mappingPath, esInfo, rtype)
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
		suffix := fmt.Sprintf("_detail_%s.go", strings.ToLower(rtype))
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
