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

// 根据字段检索查询数字等字段的topN详情

// PreDetailMatchTopNCond 使用go代码预处理渲染需要的一些逻辑，template脚本出来调试困难
func PreDetailMatchTopNCond(mappingPath string, esInfo *EsModelInfo) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 尝试加载自定义生成配置
	genCfg := LoadCustomGenConfig(mappingPath)

	// 根据配置处理全文本字段的配置
	fields := RetainTextFieldByName(esInfo.Fields, genCfg.AllTextFieldOnly, genCfg.AllTextField)

	// 根据配置文件自定义字段分组进行随机组合
	cmbFields := CombineCustom(fields, genCfg.Combine, genCfg.MaxCombine-1)
	cmbFields = LimitCombineFilter(cmbFields, map[string]int{TypeVector: -1})

	// 构造渲染模板所需的数据
	for _, cfs := range cmbFields {
		// 筛选出做聚合分析的类型的字段
		topFields := FilterOutByTypes(fields, cfs, []string{TypeNumber}, nil)

		// 过滤出配置文件指定的聚合字段
		topFields = FilterOutByName(topFields, cfs, genCfg.TermsFields, genCfg.NotTermsFields)

		// 只生成一个字段的排序，否则太多了
		topCmbs := utils.Combinations(topFields, 1)
		for _, tcmb := range topCmbs {
			names := getDetailMatchTopNFuncName(esInfo.StructName, cfs, tcmb)
			comments := getDetailMatchTopNFuncComment(esInfo.StructComment, cfs, tcmb)
			queries := getDetailMatchTopNQuery(cfs, tcmb, genCfg.TermInShould)
			for idx := range len(names) {
				ftd := &FuncTplData{
					Name:    names[idx],
					Comment: comments[idx],
					Params:  getDetailMatchTopNFuncParams(cfs),
					Query:   queries[idx],
				}
				funcDatas = append(funcDatas, ftd)
			}

		}
	}

	return funcDatas
}

// getDetailMatchTopNFuncName 获取函数名称
func getDetailMatchTopNFuncName(structName string, fields, topFields []*FieldInfo) []string {
	names := []string{}
	for _, topOpt := range TopTypes {
		fn := "Match" + structName + "By" + GenFieldsName(fields) + topOpt + GenFieldsName(topFields)
		names = append(names, fn)
	}

	return names
}

// getDetailMatchTopNFuncComment 获取函数注释
func getDetailMatchTopNFuncComment(structComment string, fields, topFields []*FieldInfo) []string {
	cmts := []string{}
	for _, topOpt := range TopTypes {
		// 函数注释
		cmt := "根据" + GenFieldsCmt(fields, true)
		cmt += "检索" + structComment + "中" + GenFieldsCmt(topFields, true) + TopNames[topOpt] + "的前N条详细数据列表\n"

		// 参数注释
		cmt += GenParamCmt(fields, false) + "// size int 前N条记录"
		cmts = append(cmts, cmt)
	}

	return cmts
}

// getDetailMatchTopNFuncParams 获取函数参数列表
func getDetailMatchTopNFuncParams(fields []*FieldInfo) string {
	fp := GenParam(fields, false) + "size int"
	return simplifyParams(fp)
}

// getDetailMatchTopNQuery 获取函数的查找条件
func getDetailMatchTopNQuery(fields, topFields []*FieldInfo, termInShould bool) []string {
	queries := []string{}
	for _, topOpt := range TopTypes {
		// match部分参数
		mq := GenMatchCond(fields)

		// term部分参数
		tq := GenTermCond(fields)

		// bool部分参数
		bq := GenBoolCond(mq, tq, termInShould)

		// sort部分参数
		sq := GenSortCond(topFields, topOpt)

		esq := GenESQueryCond(bq, "", sq, "size")

		// 拼接match和term条件
		fq := mq + tq + sq + esq
		queries = append(queries, fq)
	}

	return queries
}

// GenEsDetailMatchTopN 生成es检索详情
func GenEsDetailMatchTopN(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	// 预处理渲染所需的内容
	funcData := PreDetailMatchTopNCond(mappingPath, esInfo)
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
	outputPath = strings.Replace(outputPath, ".go", "_detail_match_topn.go", -1)
	err = os.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write output file %s: %v", outputPath, err)
	}

	// 调用go格式化工具格式化代码
	cmd := exec.Command("goimports", "-w", outputPath)
	cmd.Run()

	return nil
}
