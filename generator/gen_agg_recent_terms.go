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

// PreAggRecentTermsCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreAggRecentTermsCond(mappingPath string, esInfo *EsModelInfo, rtype string) []*FuncTplData {
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
		termsFields := FilterOutByTypes(fields, cfs, []string{TypeKeyword}, nil)

		// 过滤出配置文件指定的聚合字段
		termsFields = FilterOutByName(termsFields, cfs, genCfg.TermsFields, genCfg.NotTermsFields)

		// terms的嵌套聚合分析次序是对结果哟影响的，因此只能生成一个字段的聚合，否则太多了
		termsCmbs := utils.Combinations(termsFields, 1)
		for _, tcmb := range termsCmbs {
			names := getAggRecentTermsFuncName(esInfo.StructName, cfs, tcmb, rangeTypes, rtype, genCfg.CmpOptList)
			comments := getAggRecentTermsFuncComment(esInfo.StructComment, cfs, tcmb, rangeTypes, rtype, genCfg.CmpOptList)
			params := getAggRecentTermsFuncParams(cfs, rangeTypes, rtype, genCfg.CmpOptList)
			queries := getAggRecentTermsQuery(cfs, tcmb, rangeTypes, genCfg.TermInShould, rtype, genCfg.CmpOptList)
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

// getAggRecentTermsFuncName 获取函数名称
func getAggRecentTermsFuncName(structName string, fields, termsFields []*FieldInfo, rangeTypes []string, rtype string, optList [][]string) []string {
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
	fieldOpts := [][]string{}
	for _, f := range types {
		if getTypeMapping(f.EsFieldType) == TypeNumber {
			// 数值的多种比较
			tmps := []string{}
			for _, opts := range optList {
				tmp := f.FieldName
				for _, opt := range opts {
					tmp += opt
				}
				tmps = append(tmps, tmp)
			}
			fieldOpts = append(fieldOpts, tmps)
		} else if getTypeMapping(f.EsFieldType) == TypeDate {
			// 日期的近期查找
			tmps := []string{}
			tmp := f.FieldName
			tmp += GTE
			tmps = append(tmps, tmp)
			fieldOpts = append(fieldOpts, tmps)
		}

	}

	names := []string{}
	fn := "Terms" + GenFieldsName(termsFields) + "Of" + rtype + structName + "By" + otherName
	// 多字段之间的两两组合
	fopts := utils.Cartesian(fieldOpts)
	for _, fopt := range fopts {
		names = append(names, fn+fopt)
	}
	return names
}

// 根据发布日期大于等于和小于等于检索books表并分组统计类别的分布情况

// getAggRecentTermsFuncComment 获取函数注释
func getAggRecentTermsFuncComment(structComment string, fields, termsFields []*FieldInfo, rangeTypes []string, rtype string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	statCmt := "并分组统计"
	if len(termsFields) > 1 {
		statCmt = "并同时统计"
	}

	// 函数注释部分
	types, other := FieldFilterByTypes(fields, rangeTypes)
	otherComment := "根据"
	// 串联过滤条件的字段名
	if len(other) > 0 {
		otherComment += GenFieldsCmt(other) + "、"
	}

	fieldCmts := [][]string{}
	for _, f := range types {
		if getTypeMapping(f.EsFieldType) == TypeNumber {
			// 数值的多种比较
			tmps := []string{}
			for _, opts := range optList {
				tmp := f.FieldComment
				for _, opt := range opts {
					tmp += CmpOptNames[opt] + "和"
				}
				tmp = strings.TrimSuffix(tmp, "和")
				tmp += "、"
				tmps = append(tmps, tmp)
			}
			fieldCmts = append(fieldCmts, tmps)
		} else if getTypeMapping(f.EsFieldType) == TypeDate {
			// 日期的近期查找
			tmps := []string{}
			tmp := f.FieldComment
			tmp += RecentNames[rtype]
			tmps = append(tmps, tmp)
			fieldCmts = append(fieldCmts, tmps)
		}
	}
	funcCmts := []string{}
	fopts := utils.Cartesian(fieldCmts)
	for _, fopt := range fopts {
		fopt = strings.TrimSuffix(fopt, "、")
		funcCmts = append(funcCmts, otherComment+fopt+"检索"+structComment+statCmt+GenFieldsCmt(termsFields)+"的分布情况\n")
	}

	// 参数注释部分
	filterParam := ""
	for _, f := range other { // 过滤条件部分
		filterParam += "// " + utils.ToFirstLower(f.FieldName) + " " + f.FieldType + " " + f.FieldComment + "\n"
	}

	// 范围条件部分
	fieldParamCmts := [][]string{}
	for _, f := range types {
		if getTypeMapping(f.EsFieldType) == TypeNumber {
			tmps := []string{}
			for _, opts := range optList {
				tmp := " "
				for _, opt := range opts {
					tmp += "// " + utils.ToFirstLower(f.FieldName) + opt + " " + f.FieldType + " " + f.FieldComment + CmpOptNames[opt] + "\n"
				}
				tmps = append(tmps, tmp)
			}
			fieldParamCmts = append(fieldParamCmts, tmps)
		} else if getTypeMapping(f.EsFieldType) == TypeDate {
			tmps := []string{}
			tmp := " "
			tmp += "// " + utils.ToFirstLower(f.FieldName) + fmt.Sprintf("N%s", rtype) + " int " + f.FieldComment + RecentNames[rtype] + "\n"
			tmps = append(tmps, tmp)
			fieldParamCmts = append(fieldParamCmts, tmps)
		}
	}
	paramOpts := utils.Cartesian(fieldParamCmts)

	// 函数注释和参数注释合并
	if len(funcCmts) == len(paramOpts) {
		for idx, fc := range funcCmts {
			funcCmts[idx] = fc + filterParam + strings.TrimSuffix(paramOpts[idx], "\n")
		}
	}

	return funcCmts
}

// getAggRecentTermsFuncParams 获取函数参数列表
func getAggRecentTermsFuncParams(fields []*FieldInfo, rangeTypes []string, rtype string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}
	types, other := FieldFilterByTypes(fields, rangeTypes)
	// 过滤条件参数
	cfp := ""
	for _, f := range other {
		cfp += utils.ToFirstLower(f.FieldName) + " " + f.FieldType + ", "
	}

	// 范围条件参数
	params := [][]string{}
	for _, f := range types {
		if getTypeMapping(f.EsFieldType) == TypeNumber {
			tmps := []string{}
			for _, opts := range optList {
				tmp := ""
				for _, opt := range opts {
					tmp += utils.ToFirstLower(f.FieldName) + opt + ", "
				}
				tmp = strings.TrimSuffix(tmp, ", ")
				tmp += " " + f.FieldType + ", "
				tmps = append(tmps, tmp)
			}
			params = append(params, tmps)
		} else if getTypeMapping(f.EsFieldType) == TypeDate {
			tmps := []string{}
			tmp := ""
			tmp += utils.ToFirstLower(f.FieldName) + fmt.Sprintf("N%s", rtype) + " int, "
			tmps = append(tmps, tmp)
			params = append(params, tmps)
		}
	}

	funcParams := utils.Cartesian(params)
	for idx, fp := range funcParams {
		funcParams[idx] = simplifyParams(cfp + strings.TrimSuffix(fp, ", "))
	}
	return funcParams
}

// getAggRecentTermsQuery 获取函数的查找条件
func getAggRecentTermsQuery(fields, termsFields []*FieldInfo, rangeTypes []string, termInShould bool, rtype string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	types, other := FieldFilterByTypes(fields, rangeTypes)
	// match部分参数
	mq := GenMatchCond(other)

	// agg部分参数
	aq := GenAggWithCond(termsFields, AggFuncTerms)

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
		esq := GenESQueryCond(bq, aq)

		// 拼接match,term和agg条件
		fq := mq + tq + aq + esq

		funcRanges[idx] = fq
	}

	return funcRanges
}

// GenEsAggRecentTerms 生成es检索详情
func GenEsAggRecentTerms(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	for _, rtype := range RecentTypes {
		// 预处理渲染所需的内容
		funcData := PreAggRecentTermsCond(mappingPath, esInfo, rtype)
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
		suffix := fmt.Sprintf("_agg_%s_terms.go", strings.ToLower(rtype))
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
