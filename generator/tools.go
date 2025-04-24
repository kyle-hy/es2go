package generator

import (
	"fmt"
	"strings"

	"github.com/kyle-hy/es2go/utils"
)

// simplifyParams 函数参数列表类型的简化
func simplifyParams(paramStr string) string {
	params := strings.Split(paramStr, ",")
	typeMap := make(map[string][]string)

	for _, p := range params {
		p = strings.TrimSpace(p)
		parts := strings.Fields(p)
		if len(parts) == 2 {
			name := parts[0]
			typ := parts[1]
			typeMap[typ] = append(typeMap[typ], name)
		}
	}

	var simplified []string
	for typ, names := range typeMap {
		simplified = append(simplified, fmt.Sprintf("%s %s", strings.Join(names, ", "), typ))
	}

	return strings.Join(simplified, ", ")
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
