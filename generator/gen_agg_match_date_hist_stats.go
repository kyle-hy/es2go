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

// PreAggMatchDateHistStatsCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreAggMatchDateHistStatsCond(mappingPath string, esInfo *EsModelInfo, dhtype, stype string) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 尝试加载自定义生成配置
	genCfg := LoadCustomGenConfig(mappingPath)

	// 根据配置处理全文本字段的配置
	fields := RetainTextFieldByName(esInfo.Fields, genCfg.AllTextFieldOnly, genCfg.AllTextField)

	// 根据配置文件自定义字段分组进行随机组合
	cmbFields := CombineCustom(fields, genCfg.Combine, genCfg.MaxCombine)
	cmbFields = LimitCombineFilter(cmbFields, map[string]int{TypeVector: -1})

	// 构造渲染模板所需的数据
	for _, condCmb := range cmbFields {
		// 筛选出做聚合分析的类型的字段
		dateFields := FilterOutByTypes(fields, condCmb, []string{TypeDate}, nil)

		// 过滤出配置文件指定的聚合字段
		histFields := FilterOutByName(dateFields, condCmb, genCfg.HistFields, genCfg.NotHistFields)

		// 筛选出做聚合分析的类型的字段
		numFields := FilterOutByTypes(fields, condCmb, []string{TypeNumber}, nil)

		// 过滤出配置文件指定的分桶后再嵌套统计的字段
		histStatsFields := FilterOutByName(numFields, condCmb, genCfg.HistStatsFields, genCfg.NotHistStatsFields)

		// histogram的聚合分析
		histCmbs := utils.Combinations(histFields, 1)
		statsCmbs := utils.Combinations(histStatsFields, 1)
		for _, histCmb := range histCmbs {
			for _, statsCmb := range statsCmbs {
				ftd := &FuncTplData{
					Name:    getAggMatchDateHistStatsFuncName(esInfo.StructName, condCmb, histCmb, statsCmb, dhtype, stype),
					Comment: getAggMatchDateHistStatsFuncComment(esInfo.StructComment, condCmb, histCmb, statsCmb, dhtype, stype),
					Params:  getAggMatchDateHistStatsFuncParams(condCmb),
					Query:   getAggMatchDateHistStatsQuery(condCmb, histCmb, statsCmb, dhtype, stype),
				}
				funcDatas = append(funcDatas, ftd)
			}
		}
	}

	return funcDatas
}

// getAggMatchDateHistStatsFuncName 获取函数名称
func getAggMatchDateHistStatsFuncName(structName string, condFields, histFields, statsFields []*FieldInfo, dhtype, stype string) string {
	fn := stype + GenFieldsName(statsFields) + "In" + dhtype + "Hist" + GenFieldsName(histFields) + "Of" + structName + "By" + GenFieldsName(condFields)
	return fn
}

// getAggMatchDateHistStatsFuncComment 获取函数注释
func getAggMatchDateHistStatsFuncComment(structComment string, condFields, histFields, statsFields []*FieldInfo, dhtype, stype string) string {
	// 函数注释
	cmt := "根据" + GenFieldsCmt(condFields, true) + "检索" + structComment + "，并按" + GenFieldsCmt(histFields, true) + "分桶统计" + DateHistNames[dhtype] + GenFieldsCmt(statsFields, true) + "的" + HistStatNames[stype] + "\n"

	// 参数注释
	cmt += GenParamCmt(condFields, true)
	return cmt
}

// getAggMatchDateHistStatsFuncParams 获取函数参数列表
func getAggMatchDateHistStatsFuncParams(fields []*FieldInfo) string {
	fp := GenParam(fields, true)
	return fp
}

// getAggMatchDateHistStatsQuery 获取函数的查找条件
func getAggMatchDateHistStatsQuery(condFields, histFields, statsFields []*FieldInfo, dhtype, stype string) string {
	// match部分参数
	mq := GenMatchCond(condFields)

	// term部分参数
	tq := GenTermCond(condFields)

	// agg部分参数
	aq := GenAggWithCondOpt(histFields, AggFuncDateHist, fmt.Sprintf(AggOptCalendarInterval, DateHistInterval[dhtype]))
	aq += AddSubAggCond(statsFields, HistStatsFuncs[stype])

	// bool部分参数
	bq := GenBoolCond(mq, tq, false)
	esq := GenESQueryCond(bq, aq, "", "")

	// 拼接match和term条件
	fq := mq + tq + aq + esq

	return fq
}

// GenEsAggMatchDateHistStats 生成es检索后聚合分析
func GenEsAggMatchDateHistStats(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	for _, dhtype := range DateHistTypes {
		for _, stype := range HistStatsTypes {
			// 预处理渲染所需的内容
			funcData := PreAggMatchDateHistStatsCond(mappingPath, esInfo, dhtype, stype)
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
			suffix := fmt.Sprintf("_agg_match_hist_%s_%s.go", strings.ToLower(dhtype), strings.ToLower(stype))
			output := strings.Replace(outputPath, ".go", suffix, -1)
			err = os.WriteFile(output, buf.Bytes(), 0644)
			if err != nil {
				return fmt.Errorf("Failed to write output file %s: %v", output, err)
			}

			// 调用go格式化工具格式化代码
			cmd := exec.Command("goimports", "-w", output)
			cmd.Run()
		}
	}

	return nil
}
