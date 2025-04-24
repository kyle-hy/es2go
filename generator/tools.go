package generator

import (
	"fmt"
	"strings"

	"github.com/kyle-hy/es2go/utils"
)

// GenParamCmt 生成函数参数部分的注释
func GenParamCmt(fields []*FieldInfo) string {
	cmt := ""
	for _, f := range fields {
		cmt += "// " + utils.ToFirstLower(f.FieldName) + " " + f.FieldType + " " + f.FieldComment + "\n"
	}
	cmt = strings.TrimSuffix(cmt, "\n")
	return cmt
}

// GenFieldNames 串联参数的名称
func GenFieldNames(fields []*FieldInfo) string {
	cmt := ""
	for _, f := range fields {
		cmt += f.FieldComment + "、"
	}
	cmt = strings.TrimSuffix(cmt, "、")
	return cmt
}

// simplifyParams 函数参数列表类型的简化
func simplifyParams(paramStr string) string {
	params := strings.Split(paramStr, ",")
	var result []string

	// 当前一组的变量名和类型
	var currentNames []string
	var currentType string

	flush := func() {
		if len(currentNames) > 0 {
			result = append(result, fmt.Sprintf("%s %s", strings.Join(currentNames, ", "), currentType))
			currentNames = nil
		}
	}

	for _, p := range params {
		p = strings.TrimSpace(p)
		parts := strings.Fields(p)
		if len(parts) != 2 {
			continue // 忽略格式不正确的部分
		}
		name, typ := parts[0], parts[1]

		if typ != currentType {
			flush()
			currentType = typ
		}
		currentNames = append(currentNames, name)
	}
	flush()

	return strings.Join(result, ", ")
}

// GenParam 生成函数参数
func GenParam(fields []*FieldInfo) string {
	fp := ""
	for _, f := range fields {
		fp += utils.ToFirstLower(f.FieldName) + " " + f.FieldType + ", "
	}
	fp = strings.TrimSuffix(fp, ", ")
	return simplifyParams(fp)

}

// GenMatchCond 生成match条件
func GenMatchCond(fields []*FieldInfo) string {
	// match部分参数
	matchCnt := 0
	mq := "matches := []eq.Map{\n"
	for _, f := range fields {
		if f.EsFieldType == "text" {
			matchCnt++
			mq += fmt.Sprintf("		eq.Match(\"%s\", %s),\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName))
		}
	}
	mq += "	}\n"

	if matchCnt == 0 {
		return ""
	}
	return mq
}

// GenTermCond 生成match条件
func GenTermCond(fields []*FieldInfo) string {
	// match部分参数
	termCnt := 0
	tq := "terms := []eq.Map{\n"
	for _, f := range fields {
		if f.EsFieldType != "text" {
			termCnt++
			tq += fmt.Sprintf("		eq.Term(\"%s\", %s),\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName))
		}
	}
	tq += "	}\n"

	if termCnt == 0 {
		return ""
	}
	return tq
}

// GenBoolCond 生成bool条件
func GenBoolCond(mq, tq string, termInShould bool) string {
	// 精确条件默认放到filter中
	preciseOpt := "eq.WithFilter"
	if termInShould {
		preciseOpt = "eq.WithShould"
	}

	// 组合bool条件
	fq := "	esQuery := &eq.ESQuery{Query: eq.Bool("
	if mq != "" {
		fq += "eq.WithMust(matches)"
	}
	if tq != "" {
		if mq != "" {
			fq += fmt.Sprintf(", %s(terms)", preciseOpt)
		} else {
			fq += fmt.Sprintf("%s(terms)", preciseOpt)
		}
	}

	fq += ")}"
	return fq
}
