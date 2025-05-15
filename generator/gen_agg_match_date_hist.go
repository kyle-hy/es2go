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

// PreAggMatchDateHistCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreAggMatchDateHistCond(mappingPath string, esInfo *EsModelInfo, dhtype string) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 尝试加载自定义生成配置
	genCfg := LoadCustomGenConfig(mappingPath)

	// 根据配置处理全文本字段的配置
	fields := RetainTextFieldByName(esInfo.Fields, genCfg.AllTextFieldOnly, genCfg.AllTextField)

	// 根据配置文件自定义字段分组进行随机组合
	cmbFields := CombineCustom(fields, genCfg.Combine, genCfg.MaxCombine)
	cmbFields = LimitCombineFilter(cmbFields, map[string]int{TypeVector: -1})

	// 构造渲染模板所需的数据
	for _, cfs := range cmbFields {
		// 筛选出做聚合分析的类型的字段
		statsFields := FilterOutByTypes(fields, cfs, []string{TypeDate}, nil)

		// 过滤出配置文件指定的聚合字段
		statsFields = FilterOutByName(statsFields, cfs, genCfg.HistFields, genCfg.NotHistFields)

		// histogram的聚合分析
		statsCmbs := utils.Combinations(statsFields, 1)
		for _, scmb := range statsCmbs {
			ftd := &FuncTplData{
				Name:    getAggMatchDateHistFuncName(esInfo.StructName, cfs, scmb, dhtype),
				Comment: getAggMatchDateHistFuncComment(esInfo.StructComment, cfs, scmb, dhtype),
				Params:  getAggMatchDateHistFuncParams(cfs),
				Query:   getAggMatchDateHistQuery(cfs, scmb, dhtype),
			}
			funcDatas = append(funcDatas, ftd)
		}
	}

	return funcDatas
}

// getAggMatchDateHistFuncName 获取函数名称
func getAggMatchDateHistFuncName(structName string, fields, termsFields []*FieldInfo, dhtype string) string {
	fn := dhtype + "Hist" + GenFieldsName(termsFields) + "Of" + structName + "By" + GenFieldsName(fields)
	return fn
}

// getAggMatchDateHistFuncComment 获取函数注释
func getAggMatchDateHistFuncComment(structComment string, fields, termsFields []*FieldInfo, dhtype string) string {
	// 函数注释
	cmt := "根据" + GenFieldsCmt(fields, true) + "检索" + structComment + "并按" + GenFieldsCmt(termsFields, true) + "分桶统计" + DateHistNames[dhtype] + "的记录数量直方图分布\n"

	// 参数注释
	cmt += GenParamCmt(fields, true)
	return cmt
}

// getAggMatchDateHistFuncParams 获取函数参数列表
func getAggMatchDateHistFuncParams(fields []*FieldInfo) string {
	fp := GenParam(fields, true)
	return fp
}

// getAggMatchDateHistQuery 获取函数的查找条件
func getAggMatchDateHistQuery(fields, termsFields []*FieldInfo, dhtype string) string {
	// match部分参数
	mq := GenMatchCond(fields)

	// term部分参数
	tq := GenTermCond(fields)

	// agg部分参数
	aq := GenAggWithCondOpt(termsFields, AggFuncDateHist, fmt.Sprintf(AggOptCalendarInterval, DateHistInterval[dhtype]))

	// bool部分参数
	bq := GenBoolCond(mq, tq, false)
	esq := GenESQueryCond(bq, aq, "", "")

	// 拼接match和term条件
	fq := mq + tq + aq + esq

	return fq
}

// GenEsAggMatchDateHist 生成es检索后聚合分析
func GenEsAggMatchDateHist(mappingPath, outputPath string, esInfo *EsModelInfo) error {
	for _, dhtype := range DateHistTypes {
		// 预处理渲染所需的内容
		funcData := PreAggMatchDateHistCond(mappingPath, esInfo, dhtype)
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
		suffix := fmt.Sprintf("_agg_match_hist_%s.go", strings.ToLower(dhtype))
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
