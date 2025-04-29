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

// PreDetailRangeCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreDetailRangeCond(mappingPath string, esInfo *EsModelInfo) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 尝试加载自定义生成配置
	genCfg := LoadCustomGenConfig(mappingPath)

	// 根据配置处理全文本字段的配置
	fields := RetainTextFieldByName(esInfo.Fields, genCfg.AllTextFieldOnly, genCfg.AllTextField)

	// 根据配置文件自定义字段分组进行随机组合
	fields = FilterOutByName(fields, nil, genCfg.RangeFields, genCfg.NotRangeFields)
	cmbFields := CombineCustom(fields, genCfg.Combine, genCfg.MaxCombine)

	// 过滤出满足类型限制的组合
	cmbLimit := map[string]int{TypeNumber: 1, TypeDate: 1, TypeVector: -1}
	limitCmbs := LimitCombineFilter(cmbFields, cmbLimit)

	// 过滤出最少包含指定类型之一的组合
	rangeTypes := []string{TypeNumber, TypeDate}
	mustFileds := MustCombineFilter(limitCmbs, rangeTypes)

	// 字段随机组合
	for _, cfs := range mustFileds {
		names := getDetailRangeFuncName(esInfo.StructName, cfs, rangeTypes, genCfg.CmpOptList)
		comments := getDetailRangeFuncComment(esInfo.StructComment, cfs, rangeTypes, genCfg.CmpOptList)
		params := getDetailRangeFuncParams(cfs, rangeTypes, genCfg.CmpOptList)
		queries := getDetailRangeQuery(cfs, rangeTypes, genCfg.TermInShould, genCfg.CmpOptList)
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

// 数值比较操作
var (
	GTE        = "Gte"
	GT         = "Gt"
	LT         = "Lt"
	LTE        = "Lte"
	CmpOptList = [][]string{
		// {GTE}, {LTE},
		{GTE}, {GT}, {LT}, {LTE},
		{GTE, LTE},
		// {GTE, LT}, {GTE, LTE}, {GT, LT}, {GT, LTE},
	}
	CmpOptNames = map[string]string{
		GTE: "大于等于",
		GT:  "大于",
		LT:  "小于",
		LTE: "小于等于",
	}
)

// getDetailRangeFuncName 获取函数名称
func getDetailRangeFuncName(structName string, fields []*FieldInfo, rangeTypes []string, optList [][]string) []string {
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
		names = append(names, fn+fopt+otherName)
	}
	return names
}

// getDetailRangeFuncComment 获取函数注释
func getDetailRangeFuncComment(structComment string, fields []*FieldInfo, rangeTypes []string, optList [][]string) []string {
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

	fieldCmts := GenRangeFuncParamCmt(types, optList, nil)
	funcCmts := []string{}
	fn := "从" + structComment + "查找"
	fopts := utils.Cartesian(fieldCmts)
	for _, fopt := range fopts {
		fopt = strings.TrimSuffix(fopt, "、")
		funcCmts = append(funcCmts, otherComment+fn+fopt+"指定数值的详细数据列表和总数量\n")
	}

	// 参数注释部分
	filterParam := GenParamCmt(other, false)

	// 范围条件部分
	fieldParamCmts := GenRangeParamCmt(types, optList, nil)

	// 函数注释和参数注释合并
	paramOpts := utils.Cartesian(fieldParamCmts)
	if len(funcCmts) == len(paramOpts) {
		for idx, fc := range funcCmts {
			funcCmts[idx] = fc + filterParam + strings.TrimSuffix(paramOpts[idx], "\n")
		}
	}

	return funcCmts
}

// getDetailRangeFuncParams 获取函数参数列表
func getDetailRangeFuncParams(fields []*FieldInfo, rangeTypes []string, optList [][]string) []string {
	types, other := FieldFilterByTypes(fields, rangeTypes)
	// 过滤条件参数
	cfp := GenParam(other, false)

	// 范围条件参数
	params := GenRangeParam(types, optList, nil)

	funcParams := utils.Cartesian(params)
	for idx, fp := range funcParams {
		funcParams[idx] = simplifyParams(cfp + strings.TrimSuffix(fp, ", "))
	}
	return funcParams
}

// getDetailRangeQuery 获取函数的查找条件
func getDetailRangeQuery(fields []*FieldInfo, rangeTypes []string, termInShould bool, optList [][]string) []string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	// 精确条件默认放到filter中
	preciseOpt := "eq.WithFilter"
	if termInShould {
		preciseOpt = "eq.WithShould"
	}

	types, other := FieldFilterByTypes(fields, rangeTypes)
	// match部分参数
	matchCnt := 0
	mq := "matches := []eq.Map{\n"
	for _, f := range other {
		if f.EsFieldType == "text" {
			matchCnt++
			mq += fmt.Sprintf("		eq.Match(\"%s\", %s),\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName))
		}
	}
	mq += "	}\n"

	// match部分参数
	termCnt := 0
	tq := ""
	for _, f := range other {
		if f.EsFieldType != "text" {
			termCnt++
			tq += fmt.Sprintf("		eq.Term(\"%s\", %s),\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName))
		}
	}

	ranges := [][]string{}
	for _, f := range types {
		tmps := []string{}
		for _, opts := range optList {
			tmp := ""
			gte, gt, lt, lte := "nil", "nil", "nil", "nil"
			for _, opt := range opts {
				switch opt {
				case GTE:
					gte = utils.ToFirstLower(f.FieldName) + opt
				case GT:
					gt = utils.ToFirstLower(f.FieldName + opt)
				case LT:
					lt = utils.ToFirstLower(f.FieldName + opt)
				case LTE:
					lte = utils.ToFirstLower(f.FieldName + opt)
				}
			}
			tmp += fmt.Sprintf("		eq.Range(\"%s\", %s, %s, %s, %s),\n", f.EsFieldPath, gte, gt, lt, lte)
			tmps = append(tmps, tmp)
		}
		ranges = append(ranges, tmps)
	}

	funcRanges := utils.Cartesian(ranges)
	for idx, fq := range funcRanges {
		fq := "filters := []eq.Map{\n" + tq + fq + "	}\n"
		if matchCnt > 0 {
			fq = mq + fq
			qfmt := `	esQuery := &eq.ESQuery{Query: eq.Bool(eq.WithMust(matches), %s(filters))}`
			fq += fmt.Sprintf(qfmt, preciseOpt)
		} else {
			qfmt := `	esQuery := &eq.ESQuery{Query: eq.Bool(%s(filters))}`
			fq += fmt.Sprintf(qfmt, preciseOpt)
		}
		funcRanges[idx] = fq
	}

	return funcRanges
}

// GenEsDetailRange 生成es检索详情
func GenEsDetailRange(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	// 预处理渲染所需的内容
	funcData := PreDetailRangeCond(mappingPath, esInfo)
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
	outputPath = strings.Replace(outputPath, ".go", "_detail_range.go", -1)
	err = os.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write output file %s: %v", outputPath, err)
	}

	// 调用go格式化工具格式化代码
	cmd := exec.Command("goimports", "-w", outputPath)
	cmd.Run()

	return nil
}
