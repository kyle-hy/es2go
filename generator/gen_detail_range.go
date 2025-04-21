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

// PreDetailRangeCond 使用go代码预处理渲染需要的一些逻辑，template脚本调试困难
func PreDetailRangeCond(esInfo *EsModelInfo) []*FuncTplData {
	funcDatas := []*FuncTplData{}

	// 按数据类型分组字段
	grpFileds := GroupFieldsByType(esInfo.Fields)

	// 提取目标字段
	fields := grpFileds[TypeNumber]

	// 字段随机组合
	cmbFields := utils.Combinations(fields, 2) // 范围查询比较方式较多，限定到两个字段的组合
	for _, cfs := range cmbFields {
		names := getDetailRangeFuncName(esInfo.StructName, cfs)
		comments := getDetailRangeFuncComment(esInfo.StructName, cfs)
		params := getDetailRangeFuncParams(cfs)
		queries := getDetailRangeMatchQuery(cfs)
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
	GTE     = "Gte"
	GT      = "Gt"
	LT      = "Lt"
	LTE     = "Lte"
	optList = [][]string{
		// {GTE}, {LTE},
		{GTE}, {GT}, {LT}, {LTE},
		{GTE, LTE},
		// {GTE, LT}, {GTE, LTE}, {GT, LT}, {GT, LTE},
	}
	optNames = map[string]string{
		GTE: "大于等于",
		GT:  "大于",
		LT:  "小于",
		LTE: "小于等于",
	}
)

// getDetailRangeFuncName 获取函数名称
func getDetailRangeFuncName(structName string, fields []*FieldInfo) []string {
	fieldOpts := [][]string{}
	for _, f := range fields {
		tmps := []string{}
		for _, opts := range optList {
			tmp := f.FieldName
			for _, opt := range opts {
				tmp += opt
			}
			tmps = append(tmps, tmp)
		}
		fieldOpts = append(fieldOpts, tmps)
	}

	names := []string{}
	fn := "Range" + structName + "By"
	fopts := utils.Cartesian(fieldOpts)
	for _, fopt := range fopts {
		names = append(names, fn+fopt)
	}
	return names
}

// getDetailRangeFuncComment 获取函数注释
func getDetailRangeFuncComment(structComment string, fields []*FieldInfo) []string {
	// 函数注释部分
	fieldCmts := [][]string{}
	for _, f := range fields {
		tmps := []string{}
		for _, opts := range optList {
			tmp := f.FieldComment
			for _, opt := range opts {
				tmp += optNames[opt]
			}
			tmps = append(tmps, tmp)
		}
		fieldCmts = append(fieldCmts, tmps)
	}
	funcCmts := []string{}
	fn := "从" + structComment + "查找"
	fopts := utils.Cartesian(fieldCmts)
	for _, fopt := range fopts {
		funcCmts = append(funcCmts, fn+fopt+"指定数值的详细数据列表和总数量\n")
	}

	// 参数注释部分
	fieldParamCmts := [][]string{}
	for _, f := range fields {
		tmps := []string{}
		for _, opts := range optList {
			tmp := " "
			for _, opt := range opts {
				tmp += "// " + utils.ToFirstLower(f.FieldName) + opt + " " + f.FieldType + " " + f.FieldComment + optNames[opt] + "\n"
			}
			tmps = append(tmps, tmp)
		}
		fieldParamCmts = append(fieldParamCmts, tmps)
	}
	paramOpts := utils.Cartesian(fieldParamCmts)

	// 函数注释和参数注释合并
	if len(funcCmts) == len(paramOpts) {
		for idx, fc := range funcCmts {
			funcCmts[idx] = fc + strings.TrimSuffix(paramOpts[idx], "\n")
		}
	}

	return funcCmts
}

// getDetailRangeFuncParams 获取函数参数列表
func getDetailRangeFuncParams(fields []*FieldInfo) []string {
	params := [][]string{}
	for _, f := range fields {
		tmps := []string{}
		for _, opts := range optList {
			tmp := ""
			for _, opt := range opts {
				tmp += utils.ToFirstLower(f.FieldName) + opt + " " + f.FieldType + ", "
			}
			tmps = append(tmps, tmp)
		}
		params = append(params, tmps)
	}

	funcParams := utils.Cartesian(params)
	for idx, fp := range funcParams {
		funcParams[idx] = strings.TrimSuffix(fp, ", ")
	}
	return funcParams
}

// getDetailRangeMatchQuery 获取函数的查询条件
func getDetailRangeMatchQuery(fields []*FieldInfo) []string {
	ranges := [][]string{}
	for _, f := range fields {
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
		fq := "ranges := []eq.Map{\n" + fq + "	}\n"
		fq += `	esQuery := &eq.ESQuery{Query: eq.Bool(eq.WithFilter(ranges))}`
		funcRanges[idx] = fq
	}

	return funcRanges
}

// GenEsDetailRange 生成es检索详情
func GenEsDetailRange(outputPath string, esInfo *EsModelInfo) error {
	// 预处理渲染所需的内容
	funcData := PreDetailRangeCond(esInfo)
	detailData := DetailTplData{
		PackageName:   esInfo.PackageName,
		StructName:    esInfo.StructName,
		StructComment: esInfo.StructComment,
		IndexName:     esInfo.IndexName,
		FuncDatas:     funcData,
	}

	// 渲染
	tmpl, err := template.New("structDatail").Parse(DetailTpl)
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
