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

// PreAggMatchTermsCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreAggMatchTermsCond(mappingPath string, esInfo *EsModelInfo) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 尝试加载自定义生成配置
	genCfg := LoadCustomGenConfig(mappingPath)

	// 根据配置处理全文本字段的配置
	fields := esInfo.Fields
	if genCfg.AllTextFieldOnly && genCfg.AllTextField != "" {
		fields = RetainTextFieldByName(esInfo.Fields, genCfg.AllTextField)
	}

	// 根据配置文件自定义字段分组进行随机组合
	cmbFields := combineCustom(fields, genCfg.Combine, genCfg.MaxCombine)
	cmbFields = LimitCombineFilter(cmbFields, map[string]int{TypeVector: -1})

	// 构造渲染模板所需的数据
	for _, cfs := range cmbFields {
		// 筛选出做聚合分析的类型的字段
		termsFields := FilterOutByTypes(fields, cfs, []string{TypeKeyword}, nil)

		// 过滤出配置文件指定的聚合字段
		termsFields = FilterOutByName(termsFields, cfs, genCfg.TermsFields, genCfg.NotTermsFields)

		// terms的嵌套聚合分析次序是对结果哟影响的，因此只能生成一个字段的聚合，否则太多了
		termsCmbs := utils.Combinations(termsFields, 1)
		for _, tcmb := range termsCmbs {
			ftd := &FuncTplData{
				Name:    getAggMatchTermsFuncName(esInfo.StructName, cfs, tcmb),
				Comment: getAggMatchTermsFuncComment(esInfo.StructComment, cfs, tcmb),
				Params:  getAggMatchTermsFuncParams(cfs),
				Query:   getAggMatchTermsQuery(cfs, tcmb, genCfg.TermInShould),
			}
			funcDatas = append(funcDatas, ftd)
		}
	}

	return funcDatas
}

// getAggMatchTermsFuncName 获取函数名称
func getAggMatchTermsFuncName(structName string, fields, termsFields []*FieldInfo) string {
	fn := "Terms" + GenFieldsName(termsFields) + "Of" + structName + "By" + GenFieldsName(fields)
	return fn
}

// getAggMatchTermsFuncComment 获取函数注释
func getAggMatchTermsFuncComment(structComment string, fields, termsFields []*FieldInfo) string {
	statCmt := "并分组统计"
	if len(termsFields) > 1 {
		statCmt = "并同时统计"
	}
	// 函数注释
	cmt := "根据" + GenFieldsCmt(fields) + "检索" + structComment + statCmt + GenFieldsCmt(termsFields) + "的分布情况\n"

	// 参数注释
	cmt += GenParamCmt(fields)

	return cmt
}

// getAggMatchTermsFuncParams 获取函数参数列表
func getAggMatchTermsFuncParams(fields []*FieldInfo) string {
	fp := GenParam(fields)
	return fp
}

// getAggMatchTermsQuery 获取函数的查找条件
func getAggMatchTermsQuery(fields, termsFields []*FieldInfo, termInShould bool) string {
	// match部分参数
	mq := GenMatchCond(fields)

	// term部分参数
	tq := GenTermCond(fields)

	// agg部分参数
	aq := GenAggWithCond(termsFields, AggFuncTerms)

	// bool部分参数
	bq := GenBoolCond(mq, tq, termInShould)
	esq := GenESQueryCond(bq, aq)

	// 拼接match和term条件
	fq := mq + tq + aq + esq

	return fq
}

// GenEsAggMatchTerms 生成es检索后聚合分析
func GenEsAggMatchTerms(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	// 预处理渲染所需的内容
	funcData := PreAggMatchTermsCond(mappingPath, esInfo)
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
	outputPath = strings.Replace(outputPath, ".go", "_agg_match_terms.go", -1)
	err = os.WriteFile(outputPath, buf.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("Failed to write output file %s: %v", outputPath, err)
	}

	// 调用go格式化工具格式化代码
	cmd := exec.Command("goimports", "-w", outputPath)
	cmd.Run()

	return nil
}
