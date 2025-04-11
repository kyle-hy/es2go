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
		queries := getDetailRangeMatchQuery(cfs)
		utils.JPrint(comments)
		for idx := range len(names) {
			ftd := &FuncTplData{
				Name:    names[idx],
				Comment: comments[idx],
				Params:  getDetailRangeFuncParams(cfs),
				Query:   queries[idx],
			}
			funcDatas = append(funcDatas, ftd)
		}
	}
	utils.JPrint(funcDatas)

	return funcDatas
}

// 数值比较操作
var (
	GTE     = "Gte"
	GT      = "Gt"
	LT      = "Lt"
	LTE     = "Lte"
	optList = [][]string{
		{GTE}, {GT}, {LT}, {LTE},
		{GTE, LT}, {GTE, LTE}, {GT, LT}, {GT, LTE},
		{LT, GTE}, {LTE, GTE}, {LT, GT}, {LTE, GT},
	}
	optNames = map[string]string{
		GTE: "大于等于",
		GT:  "大于",
		LT:  "小于",
		LTE: "小于等于",
	}
)

// getDetailRangeFuncName 获取函数名称
// gte, gt, lt, lte
func getDetailRangeFuncName(structName string, fields []*FieldInfo) []string {
	names := []string{}
	for _, opts := range optList {
		if len(fields) == len(opts) {
			fn := "Query" + structName + "By"
			for idx, opt := range opts {
				fn += fields[idx].FieldName + opt
			}
			names = append(names, fn)
		}
	}
	return names
}

// getDetailRangeFuncComment 获取函数注释
func getDetailRangeFuncComment(structComment string, fields []*FieldInfo) []string {
	comments := []string{}
	for _, opts := range optList {
		if len(fields) == len(opts) {
			// 函数注释
			cmt := "查找"
			for idx, opt := range opts {
				optName := optNames[opt]
				cmt += fields[idx].FieldComment + optName + "、"
			}
			cmt = strings.TrimSuffix(cmt, "、")
			cmt += "指定数值的" + structComment + "详细数据"

			// 参数注释
			for _, f := range fields {
				cmt += "\n// " + utils.ToFirstLower(f.FieldName) + " " + f.FieldType + " " + f.FieldComment
			}
			comments = append(comments, cmt)
		}
	}

	return comments
}

// getDetailRangeFuncParams 获取函数参数列表
func getDetailRangeFuncParams(fields []*FieldInfo) string {
	fp := ""
	for _, f := range fields {
		fp += utils.ToFirstLower(f.FieldName) + " " + f.FieldType + ", "
	}
	fp = strings.TrimSuffix(fp, ", ")
	return fp
}

// getDetailRangeMatchQuery 获取函数的查询条件
func getDetailRangeMatchQuery(fields []*FieldInfo) []string {
	fqs := []string{}
	for _, opts := range optList {
		if len(fields) == len(opts) {
			if len(fields) == 1 { // 单条件查询
				f := fields[0]
				opt := opts[0]
				gte, gt, lt, lte := "nil", "nil", "nil", "nil"
				switch opt {
				case GTE:
					gte = utils.ToFirstLower(f.FieldName)
				case GT:
					gt = utils.ToFirstLower(f.FieldName)
				case LT:
					lt = utils.ToFirstLower(f.FieldName)
				case LTE:
					lte = utils.ToFirstLower(f.FieldName)
				}

				fq := "esQuery := &eq.ESQuery{\n"
				fq += fmt.Sprintf("		Query: eq.Range(\"%s\", %s, %s, %s, %s),\n", f.EsFieldPath, gte, gt, lt, lte)
				fq += "	}\n"
				fqs = append(fqs, fq)
			} else { // 多条件查询
				fq := "matches := []eq.Map{\n"
				for idx, opt := range opts {
					f := fields[idx]
					gte, gt, lt, lte := "nil", "nil", "nil", "nil"
					switch opt {
					case GTE:
						gte = utils.ToFirstLower(f.FieldName)
					case GT:
						gt = utils.ToFirstLower(f.FieldName)
					case LT:
						lt = utils.ToFirstLower(f.FieldName)
					case LTE:
						lte = utils.ToFirstLower(f.FieldName)
					}
					fq += fmt.Sprintf("		eq.Range(\"%s\", %s, %s, %s, %s),\n", f.EsFieldPath, gte, gt, lt, lte)
				}
				fq += "	}\n"

				fq += `	esQuery := &eq.ESQuery{Query: eq.Bool(eq.WithMust(matches))}`
				fqs = append(fqs, fq)
			}
		}
	}
	return fqs
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
