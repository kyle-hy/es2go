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

// PreAggMatchStatsCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreAggMatchStatsCond(mappingPath string, esInfo *EsModelInfo, stype string) []*FuncTplData {
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
		statsFields := FilterOutFields(fields, cfs, []string{TypeNumber}, nil)

		// terms的嵌套聚合分析次序是对结果哟影响的，因此只能生成一个字段的聚合，否则太多了
		statsCmbs := utils.Combinations(statsFields, 1)
		for _, scmb := range statsCmbs {
			ftd := &FuncTplData{
				Name:    getAggMatchStatsFuncName(esInfo.StructName, cfs, scmb, stype),
				Comment: getAggMatchStatsFuncComment(esInfo.StructComment, cfs, scmb, stype),
				Params:  getAggMatchStatsFuncParams(cfs),
				Query:   getAggMatchStatsQuery(cfs, scmb, stype),
			}
			funcDatas = append(funcDatas, ftd)
		}
	}

	return funcDatas
}

// getAggMatchStatsFuncName 获取函数名称
func getAggMatchStatsFuncName(structName string, fields, termsFields []*FieldInfo, stype string) string {
	fn := stype + GenFieldsName(termsFields) + "Of" + structName + "By" + GenFieldsName(fields)
	return fn
}

// getAggMatchStatsFuncComment 获取函数注释
func getAggMatchStatsFuncComment(structComment string, fields, termsFields []*FieldInfo, stype string) string {
	// 函数注释
	cmt := "根据" + GenFieldsCmt(fields) + "检索" + structComment + "并计算" + GenFieldsCmt(termsFields) + "的" + StatNames[stype] + "\n"

	// 参数注释
	cmt += GenParamCmt(fields)
	return cmt
}

// getAggMatchStatsFuncParams 获取函数参数列表
func getAggMatchStatsFuncParams(fields []*FieldInfo) string {
	fp := GenParam(fields)
	return fp
}

// getAggMatchStatsQuery 获取函数的查找条件
func getAggMatchStatsQuery(fields, termsFields []*FieldInfo, stype string) string {
	// match部分参数
	mq := GenMatchCond(fields)

	// term部分参数
	tq := GenTermCond(fields)

	// agg部分参数
	aq := GenAggWithCond(termsFields, StatsFuncs[stype])
	// aq := GenAggNestedCond(termsFields, AggTypeTerms)

	// bool部分参数
	bq := GenBoolCond(mq, tq, false)
	esq := GenESQueryCond(bq, aq)

	// 拼接match和term条件
	fq := mq + tq + aq + esq

	return fq
}

// GenEsAggMatchStats 生成es检索后聚合分析
func GenEsAggMatchStats(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	for _, stype := range StatsTypes {
		// 预处理渲染所需的内容
		funcData := PreAggMatchStatsCond(mappingPath, esInfo, stype)
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
		suffix := fmt.Sprintf("_agg_match_%s.go", strings.ToLower(stype))
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
