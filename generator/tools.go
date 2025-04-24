package generator

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/kyle-hy/es2go/utils"
)

// FilterOutFields 提取fields中剩余的元素
// mustTypes 要提取的类型,不能同时传
// notTypes 要排除的类型,不能同时传
func FilterOutFields(fields, cmb []*FieldInfo, mustTypes, notTypes []string) []*FieldInfo {
	left := []*FieldInfo{}
	for _, f := range fields {
		found := false
		for _, c := range cmb {
			if f.EsFieldPath == c.EsFieldPath {
				found = true
				break
			}
		}

		t := getTypeMapping(f.EsFieldType)
		if !found {
			if (len(mustTypes) == 0 && len(notTypes) == 0) ||
				(len(mustTypes) > 0 && slices.Contains(mustTypes, t)) ||
				(len(notTypes) > 0 && !slices.Contains(notTypes, t)) {
				left = append(left, f)
			}
		}
	}
	return left
}

// GenParamCmt 生成函数参数部分的注释
func GenParamCmt(fields []*FieldInfo) string {
	cmt := ""
	for _, f := range fields {
		cmt += "// " + utils.ToFirstLower(f.FieldName) + " " + f.FieldType + " " + f.FieldComment + "\n"
	}
	cmt = strings.TrimSuffix(cmt, "\n")
	return cmt
}

// GenFieldsCmt 串联参数列表的注释
func GenFieldsCmt(fields []*FieldInfo) string {
	cmt := ""
	for _, f := range fields {
		cmt += f.FieldComment + "、"
	}
	cmt = strings.TrimSuffix(cmt, "、")
	return cmt
}

// GenFieldsName 串联参数列表的名称
func GenFieldsName(fields []*FieldInfo) string {
	cmt := ""
	for _, f := range fields {
		cmt += f.FieldName
	}
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

// 聚合方式枚举
const (
	AggTypeTerms = "terms"
)

// GenAggWithCond 生成多个字段同时聚合
func GenAggWithCond(fields []*FieldInfo, aggType string) string {
	aq := ""
	suffix := ""
	prefix := "eq.TermsAgg"
	for idx, f := range fields {
		if idx > 0 {
			prefix = ".With(eq.TermsAgg"
			suffix = ")"
		}
		switch aggType {
		case AggTypeTerms:
			aq += fmt.Sprintf("%s(\"%s\")%s", prefix, utils.ToFirstLower(f.FieldName), suffix)
		}
	}
	if len(fields) > 0 {
		return "	aggs :=" + aq + "\n"
	}
	return ""
}

// GenAggNestedCond 生成多个字段嵌套聚合
func GenAggNestedCond(fields []*FieldInfo, aggType string) string {
	// aggs := eq.TermsAgg("zg").Nested(eq.TermsAgg("ak").Nested(eq.TermsAgg("bk")))
	aq := ""
	suffix := ""
	prefix := "eq.TermsAgg"
	for idx, f := range fields {
		if idx > 0 {
			prefix = ".Nested(eq.TermsAgg"
			suffix += ")"
		}
		switch aggType {
		case AggTypeTerms:
			aq += fmt.Sprintf("%s(\"%s\")", prefix, utils.ToFirstLower(f.FieldName))
		}
	}
	if len(fields) > 0 {
		return "	aggs :=" + aq + suffix + "\n"
	}
	return ""
}

// GenTermCond 生成match条件
func GenTermCond(fields []*FieldInfo) string {
	// match部分参数
	termCnt := 0
	tq := "	terms := []eq.Map{\n"
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
	fq := "eq.Bool("
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

	fq += ")"
	return fq
}

// GenESQueryCond 生成bool条件
func GenESQueryCond(query, agg string) string {
	// 组合bool条件
	fq := "	esQuery := &eq.ESQuery{Query: " + query
	if agg != "" {
		fq += ", Agg: aggs"
	}
	fq += "}"
	return fq
}

// IsEmptySlice 判断多重嵌套切片是否为空
func IsEmptySlice(data any) bool {
	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Slice {
		return false
	}

	for i := range val.Len() {
		elem := val.Index(i)

		// 如果是 interface，要取出真正的值
		for elem.Kind() == reflect.Interface {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}

		if elem.Kind() == reflect.Slice {
			if !IsEmptySlice(elem.Interface()) {
				return false
			}
		} else if elem.IsValid() && !IsZero(elem) {
			return false
		}
	}

	return true
}

// IsZero 判断是否为默认零值
func IsZero(v reflect.Value) bool {
	zero := reflect.Zero(v.Type()).Interface()
	return reflect.DeepEqual(v.Interface(), zero)
}

// combineCustom 根据指定列表随机组合数组的元素
// list 字段分组，相当于将宽表拆成多个少字段的表，减少combine组合数
func combineCustom(items []*FieldInfo, list [][]string, maxCombine int) [][]*FieldInfo {
	var all [][]*FieldInfo
	keyDict := map[string]int{}
	if maxCombine <= 0 {
		maxCombine = MaxCombine
	}

	// 如果配置为空，则使用全部字段
	if IsEmptySlice(list) {
		return utils.Combinations(items, maxCombine)
	}

	// 过滤出字段
	for _, names := range list {
		fields := []*FieldInfo{}
		for _, n := range names {
			for _, i := range items {
				if i.EsFieldPath == n {
					fields = append(fields, i)
				}
			}
		}
		cmbs := utils.Combinations(fields, maxCombine)

		// 过滤掉重复的组合
		for _, cmb := range cmbs {
			key := getFieldsKey(cmb)
			if _, ok := keyDict[key]; ok {
				// 已存在组合，跳过
				continue
			} else {
				keyDict[key] = 1
				all = append(all, cmb)
			}
		}
	}
	return all
}

// getFieldsKey 将字段名排序后串联为key
func getFieldsKey(fields []*FieldInfo) string {
	ks := make([]string, len(fields))
	for _, f := range fields {
		ks = append(ks, f.EsFieldPath)
	}
	sort.Strings(ks)
	return strings.Join(ks, "")
}
