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

// PreDetailRangeTopNCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreDetailRangeTopNCond(mappingPath string, esInfo *EsModelInfo) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 尝试加载自定义生成配置
	genCfg := LoadCustomGenConfig(mappingPath)

	// 根据配置处理全文本字段的配置
	fields := RetainTextFieldByName(esInfo.Fields, genCfg.AllTextFieldOnly, genCfg.AllTextField)

	// 根据配置文件自定义字段分组进行随机组合
	fields = FilterOutByName(fields, nil, genCfg.RangeFields, genCfg.NotRangeFields)
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
		topFields := FilterOutByTypes(fields, cfs, []string{TypeNumber}, nil)

		// 过滤出配置文件指定的聚合字段
		topFields = FilterOutByName(topFields, cfs, genCfg.TopNFields, genCfg.NotTopNFields)

		// 只生成一个字段的排序，否则太多了
		topCmbs := utils.Combinations(topFields, 1)
		for _, tcmb := range topCmbs {
			names := getDetailRangeTopNFuncName(esInfo.StructName, cfs, tcmb, rangeTypes, genCfg.CmpOptList)
			comments := getDetailRangeTopNFuncComment(esInfo.StructComment, cfs, tcmb, rangeTypes, genCfg.CmpOptList)
			params := getDetailRangeTopNFuncParams(cfs, rangeTypes, genCfg.CmpOptList)
			queries := getDetailRangeTopNQuery(cfs, tcmb, rangeTypes, genCfg.TermInShould, genCfg.CmpOptList)
			for idx := range len(names) {
				ftd := &FuncTplData{
					Name:    names[idx],
					Comment: comments[idx],
					Params:  params[idx/2],
					Query:   queries[idx],
				}
				funcDatas = append(funcDatas, ftd)
			}
		}
	}

	return funcDatas
}

// getDetailRangeTopNFuncName 获取函数名称
func getDetailRangeTopNFuncName(structName string, fields, topFields []*FieldInfo, rangeTypes []string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	types, other := FieldFilterByTypes(fields, rangeTypes)
	otherName := ""
	// 串联过滤条件的字段名
	if len(other) > 0 {
		otherName = "With" + GenFieldsName(other)
	}

	// 各字段与比较符号列表的串联
	fieldOpts := GenRangeFieldName(types, optList, nil)

	names := []string{}
	fn := "Range" + structName + "By"
	// 多字段之间的两两组合
	fopts := utils.Cartesian(fieldOpts)
	for _, fopt := range fopts {
		for _, topOpt := range TopTypes {
			names = append(names, fn+fopt+otherName+topOpt+GenFieldsName(topFields))
		}
	}
	return names
}

// getDetailRangeTopNFuncComment 获取函数注释
func getDetailRangeTopNFuncComment(structComment string, fields, topFields []*FieldInfo, rangeTypes []string, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	// 函数注释部分
	types, other := FieldFilterByTypes(fields, rangeTypes)
	otherComment := "根据" + GenFieldsCmt(other, false)
	fieldCmts := GenRangeFuncParamCmt(types, optList, nil)
	funcCmts := []string{}
	fn := "检索" + structComment + "中"
	fopts := utils.Cartesian(fieldCmts)
	for _, fopt := range fopts {
		fopt = strings.TrimSuffix(fopt, "、")
		for _, topOpt := range TopTypes {
			funcCmts = append(funcCmts, otherComment+fopt+fn+GenFieldsCmt(topFields, true)+TopNames[topOpt]+"的前N条详细数据列表\n")
		}
	}

	// 参数注释部分
	filterParam := GenParamCmt(other, false)

	// 范围条件部分
	fieldParamCmts := GenRangeParamCmt(types, optList, nil)

	// 函数注释和参数注释合并
	paramOpts := utils.Cartesian(fieldParamCmts)
	if len(funcCmts) == 2*len(paramOpts) {
		for idx, fc := range funcCmts {
			funcCmts[idx] = fc + filterParam + paramOpts[idx/2] + "// size int 前N条记录"
		}
	}

	return funcCmts
}

// getDetailRangeTopNFuncParams 获取函数参数列表
func getDetailRangeTopNFuncParams(fields []*FieldInfo, rangeTypes []string, optList [][]string) []string {
	types, other := FieldFilterByTypes(fields, rangeTypes)
	// 过滤条件参数
	cfp := GenParam(other, false)

	// 范围条件参数
	params := GenRangeParam(types, optList, nil)

	funcParams := utils.Cartesian(params)
	for idx, fp := range funcParams {
		funcParams[idx] = simplifyParams(cfp + fp + "size int")
	}
	return funcParams
}

// getDetailRangeTopNQuery 获取函数的查找条件
func getDetailRangeTopNQuery(fields, topFields []*FieldInfo, rangeTypes []string, termInShould bool, optList [][]string) []string {
	types, other := FieldFilterByTypes(fields, rangeTypes)

	// match部分参数map
	mq := GenMatchCond(other)

	// term部分条件
	eqt := eqTerms(other)

	ranges := eqRanges(types, optList, nil)
	funcRanges := utils.Cartesian(ranges)

	queries := []string{}
	for _, fq := range funcRanges {
		for _, topOpt := range TopTypes {
			tq := WrapTermCond(eqt + fq)
			bq := GenBoolCond(mq, tq, termInShould)
			sq := GenSortCond(topFields, topOpt) // sort部分参数
			esq := GenESQueryCond(bq, "", sq, "size")
			queries = append(queries, mq+tq+sq+esq)
		}
	}

	return queries
}

// GenEsDetailRangeTopN 生成es检索详情
func GenEsDetailRangeTopN(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	// 预处理渲染所需的内容
	funcData := PreDetailRangeTopNCond(mappingPath, esInfo)
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
	outputPath = strings.Replace(outputPath, ".go", "_detail_range_topn.go", -1)
	err = os.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write output file %s: %v", outputPath, err)
	}

	// 调用go格式化工具格式化代码
	cmd := exec.Command("goimports", "-w", outputPath)
	cmd.Run()

	return nil
}
