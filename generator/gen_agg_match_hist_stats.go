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

// PreAggMatchHistStatsCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreAggMatchHistStatsCond(mappingPath string, esInfo *EsModelInfo, stype string) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 尝试加载自定义生成配置
	genCfg := LoadCustomGenConfig(mappingPath)

	// 根据配置处理全文本字段的配置
	fields := RetainTextFieldByName(esInfo.Fields, genCfg.AllTextFieldOnly, genCfg.AllTextField)

	// 根据配置文件自定义字段分组进行随机组合
	fieldCmds := CombineCustom(fields, genCfg.Combine, genCfg.MaxCombine)
	fieldCmds = LimitCombineFilter(fieldCmds, map[string]int{TypeVector: -1})

	// 构造渲染模板所需的数据
	for _, condCmb := range fieldCmds {
		// 筛选出做聚合分析的类型的字段
		numFields := FilterOutByTypes(fields, condCmb, []string{TypeNumber}, nil)

		// 过滤出配置文件指定的分桶聚合字段
		histFields := FilterOutByName(numFields, condCmb, genCfg.HistFields, genCfg.NotHistFields)

		// 过滤出配置文件指定的分桶后再嵌套统计的字段
		histStatsFields := FilterOutByName(numFields, condCmb, genCfg.HistStatsFields, genCfg.NotHistStatsFields)

		// histogram的聚合分析
		histCmbs := utils.Combinations(histFields, 1)
		statsCmbs := utils.Combinations(histStatsFields, 1)
		for _, histCmb := range histCmbs {
			for _, statsCmb := range statsCmbs {
				ftd := &FuncTplData{
					Name:    getAggMatchHistStatsFuncName(esInfo.StructName, condCmb, histCmb, statsCmb, stype),
					Comment: getAggMatchHistStatsFuncComment(esInfo.StructComment, condCmb, histCmb, statsCmb, stype),
					Params:  getAggMatchHistStatsFuncParams(condCmb),
					Query:   getAggMatchHistStatsQuery(condCmb, histCmb, statsCmb, stype),
				}
				funcDatas = append(funcDatas, ftd)
			}
		}
	}

	return funcDatas
}

// getAggMatchHistStatsFuncName 获取函数名称
func getAggMatchHistStatsFuncName(structName string, condFields, histFields, statsFields []*FieldInfo, stype string) string {
	fn := stype + GenFieldsName(statsFields) + "In" + "Hist" + GenFieldsName(histFields) + "Of" + structName + "By" + GenFieldsName(condFields)
	return fn
}

// getAggMatchHistStatsFuncComment 获取函数注释
func getAggMatchHistStatsFuncComment(structComment string, condFields, histFields, statsFields []*FieldInfo, stype string) string {
	// 函数注释
	cmt := "根据" + GenFieldsCmt(condFields, true) + "检索" + structComment + "，并按" + GenFieldsCmt(histFields, true) + "区间分桶统计" + GenFieldsCmt(statsFields, true) + "的" + HistStatNames[stype] + "\n"
	// 参数注释
	cmt += GenParamCmt(condFields, false)
	cmt += "// histInterval float64 分桶聚合的" + GenFieldsCmt(histFields, true) + "区间间隔"
	return cmt
}

// getAggMatchHistStatsFuncParams 获取函数参数列表
func getAggMatchHistStatsFuncParams(fields []*FieldInfo) string {
	fp := GenParam(fields, false)
	fp += "histInterval float64"
	return fp
}

// getAggMatchHistStatsQuery 获取函数的查找条件
func getAggMatchHistStatsQuery(condFields, histFields, statsFields []*FieldInfo, stype string) string {
	// match部分参数
	mq := GenMatchCond(condFields)

	// term部分参数
	tq := GenTermCond(condFields)

	// agg部分参数
	aq := GenAggWithCondOpt(histFields, AggFuncHist, AggOptInterval)
	aq += GenAddNestedAgg(statsFields, HistStatsFuncs[stype])

	// bool部分参数
	bq := GenBoolCond(mq, tq, false)
	esq := GenESQueryCond(bq, aq, "", "")

	// 拼接match和term条件
	fq := mq + tq + aq + esq
	return fq
}

// GenEsAggMatchHistStats 生成es检索后聚合分析
func GenEsAggMatchHistStats(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	for _, stype := range HistStatsTypes {
		// 预处理渲染所需的内容
		funcData := PreAggMatchHistStatsCond(mappingPath, esInfo, stype)
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
		suffix := fmt.Sprintf("_agg_match_hist_%s.go", strings.ToLower(stype))
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
