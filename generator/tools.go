package generator

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/kyle-hy/es2go/utils"
)

// FilterOutByTypes 根据子字段列表及指定的类型，过滤出剩余的字段
// mustTypes 要提取的类型,不能同时传
// notTypes 要排除的类型,不能同时传
func FilterOutByTypes(fields, cmb []*FieldInfo, mustTypes, notTypes []string) []*FieldInfo {
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

// FilterOutByName 根据子字段列表及指定的名称，过滤出剩余的字段
// mustTypes 要提取的类型,不能同时传
// notTypes 要排除的类型,不能同时传
func FilterOutByName(fields, cmb []*FieldInfo, mustNames, notNames []string) []*FieldInfo {
	left := []*FieldInfo{}
	for _, f := range fields {
		found := false
		for _, c := range cmb {
			if f.EsFieldPath == c.EsFieldPath {
				found = true
				break
			}
		}

		if !found {
			if (len(mustNames) == 0 && len(notNames) == 0) ||
				(len(mustNames) > 0 && slices.Contains(mustNames, f.EsFieldPath)) ||
				(len(notNames) > 0 && !slices.Contains(notNames, f.EsFieldPath)) {
				left = append(left, f)
			}
		}
	}
	return left
}

// GenParamCmt 生成函数参数部分的注释
func GenParamCmt(fields []*FieldInfo, trimSuffix bool) string {
	cmt := ""
	for _, f := range fields {
		cmt += "// " + utils.ToFirstLower(f.FieldName) + " " + f.FieldType + " " + f.FieldComment + "\n"
	}
	if trimSuffix {
		cmt = strings.TrimSuffix(cmt, "\n")
	}
	return cmt
}

// GenRangeParamCmt 生成函数范围查询参数部分的注释
func GenRangeParamCmt(fields []*FieldInfo, optList [][]string, limitTypes []string) [][]string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	// 范围条件部分
	fieldParamCmts := [][]string{}
	for _, f := range fields {
		// 如果存在类型限制
		if len(limitTypes) > 0 && !slices.Contains(limitTypes, getTypeMapping(f.EsFieldType)) {
			continue
		}

		tmps := []string{}
		for _, opts := range optList {
			tmp := " "
			for _, opt := range opts {
				tmp += "// " + utils.ToFirstLower(f.FieldName) + opt + " " + f.FieldType + " " + f.FieldComment + CmpOptNames[opt] + "\n"
			}
			tmps = append(tmps, tmp)
		}
		fieldParamCmts = append(fieldParamCmts, tmps)
	}
	return fieldParamCmts
}

// GenRangeFuncParamCmt 生成范围查询函数名的参数部分的注释
func GenRangeFuncParamCmt(fields []*FieldInfo, optList [][]string, limitTypes []string) [][]string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	// 范围条件部分
	fieldCmts := [][]string{}
	for _, f := range fields {
		// 如果存在类型限制
		if len(limitTypes) > 0 && !slices.Contains(limitTypes, getTypeMapping(f.EsFieldType)) {
			continue
		}

		tmps := []string{}
		for _, opts := range optList {
			tmp := f.FieldComment
			for _, opt := range opts {
				tmp += CmpOptNames[opt] + "和"
			}
			tmp = strings.TrimSuffix(tmp, "和")
			tmp += "、"
			tmps = append(tmps, tmp)
		}
		fieldCmts = append(fieldCmts, tmps)
	}
	return fieldCmts
}

// GenRecentFuncParamCmt 生成近期查询参数名的参数部分注释
func GenRecentFuncParamCmt(fields []*FieldInfo, rtype string, optList [][]string, limitTypes []string) [][]string {
	fieldCmts := [][]string{}
	for _, f := range fields {
		// 如果存在类型限制
		if len(limitTypes) > 0 && !slices.Contains(limitTypes, getTypeMapping(f.EsFieldType)) {
			continue
		}
		tmps := []string{}
		tmp := f.FieldComment
		tmp += RecentNames[rtype]
		tmps = append(tmps, tmp)
		fieldCmts = append(fieldCmts, tmps)
	}
	return fieldCmts
}

// GenFieldsCmt 串联参数列表的注释
func GenFieldsCmt(fields []*FieldInfo, trimSuffix bool) string {
	cmt := ""
	for _, f := range fields {
		cmt += f.FieldComment + "、"
	}
	if trimSuffix {
		cmt = strings.TrimSuffix(cmt, "、")
	}
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

func joinParams(ps [][]string) string {
	tmp := []string{}
	for _, p := range ps {
		tmp = append(tmp, strings.Join(p, " "))
	}
	return strings.Join(tmp, ", ")
}

// simplifyParams 函数参数列表类型的简化
func simplifyParams(paramStr string) string {
	tmpSlice := [][]string{}
	params := strings.Split(paramStr, ",")
	for _, param := range params {
		fields := strings.Fields(param)
		tmpSlice = append(tmpSlice, fields)
	}

	if len(tmpSlice) <= 1 {
		return joinParams(tmpSlice)
	}

	simply := [][]string{}
	for idx := range len(tmpSlice) {
		p1 := tmpSlice[idx]
		if idx+1 >= len(tmpSlice) {
			simply = append(simply, p1)
			break
		}

		p2 := tmpSlice[idx+1]
		if len(p1) == len(p2) && len(p1) == 2 {
			if p1[1] == p2[1] {
				simply = append(simply, p1[:1])
			} else {
				simply = append(simply, p1)
			}
		} else {
			simply = append(simply, p1)
		}
	}

	return joinParams(simply)
}

// GenParam 生成函数参数
func GenParam(fields []*FieldInfo, trimSuffix bool) string {
	fp := ""
	for _, f := range fields {
		fp += utils.ToFirstLower(f.FieldName) + " " + f.FieldType + ", "
	}
	if trimSuffix {
		fp = strings.TrimSuffix(fp, ", ")
	}
	return simplifyParams(fp)

}

// GenRecentParam 生成近期查询参数
func GenRecentParam(fields []*FieldInfo, rtype string, optList [][]string, limitTypes []string) [][]string {
	params := [][]string{}
	for _, f := range fields {
		// 如果存在类型限制
		if len(limitTypes) > 0 && !slices.Contains(limitTypes, getTypeMapping(f.EsFieldType)) {
			continue
		}
		tmps := []string{}
		tmp := ""
		tmp += utils.ToFirstLower(f.FieldName) + fmt.Sprintf("N%s", rtype) + " int, "
		tmps = append(tmps, tmp)
		params = append(params, tmps)
	}
	return params
}

// GenRecentParamCmt 生成近期查询参数的注释
func GenRecentParamCmt(fields []*FieldInfo, rtype string, optList [][]string, limitTypes []string) [][]string {
	params := [][]string{}
	for _, f := range fields {
		// 如果存在类型限制
		if len(limitTypes) > 0 && !slices.Contains(limitTypes, getTypeMapping(f.EsFieldType)) {
			continue
		}
		tmps := []string{}
		tmp := "// " + utils.ToFirstLower(f.FieldName) + fmt.Sprintf("N%s", rtype) + " int " + f.FieldComment + RecentNames[rtype] + "\n"
		tmps = append(tmps, tmp)
		params = append(params, tmps)
	}
	return params
}

// GenRangeParam 生成函数氛围查询参数
func GenRangeParam(fields []*FieldInfo, optList [][]string, limitTypes []string) [][]string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	// 范围条件参数
	params := [][]string{}
	for _, f := range fields {
		// 如果存在类型限制
		if len(limitTypes) > 0 && !slices.Contains(limitTypes, getTypeMapping(f.EsFieldType)) {
			continue
		}
		tmps := []string{}
		for _, opts := range optList {
			tmp := ""
			for _, opt := range opts {
				tmp += utils.ToFirstLower(f.FieldName) + opt + ", "
			}
			tmp = strings.TrimSuffix(tmp, ", ")
			tmp += " " + f.FieldType + ", "
			tmps = append(tmps, tmp)
		}
		params = append(params, tmps)
	}
	return params
}

// GenRangeFieldName 生成函数名称的字段名部分
func GenRangeFieldName(fields []*FieldInfo, optList [][]string, limitTypes []string) [][]string {
	if len(optList) == 0 {
		optList = CmpOptList
	}

	// 范围条件参数
	fieldOpts := [][]string{}
	for _, f := range fields {
		// 如果存在类型限制
		if len(limitTypes) > 0 && !slices.Contains(limitTypes, getTypeMapping(f.EsFieldType)) {
			continue
		}
		// 数值的多种比较
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
	return fieldOpts
}

// GenMatchCond 生成match条件
func GenMatchCond(fields []*FieldInfo) string {
	eqm := eqMatches(fields)
	if eqm != "" {
		return "matches := []eq.Map{\n" + eqm + "}\n"
	}
	return ""
}

// GenFilterCond 生成Filter条件
func GenFilterCond(fields []*FieldInfo) string {
	// match部分参数
	filterCnt := 0
	fq := "matches := []eq.Map{\n"
	for _, f := range fields {
		filterCnt++
		if f.EsFieldType != "text" {
			fq += fmt.Sprintf("		eq.Term(\"%s\", %s),\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName))
		}
		if f.EsFieldType == "text" {
			fq += fmt.Sprintf("		eq.Match(\"%s\", %s),\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName))
		}
	}
	fq += "	}\n"

	if filterCnt == 0 {
		return ""
	}
	return fq
}

// GenTermRangeCond 生成term与range随机组合的条件
func GenTermRangeCond(fields, rangeFields []*FieldInfo, optList [][]string) []string {
	// term部分参数
	terms := eqTerms(fields)

	// 范围比较部分
	ranges := eqRanges(rangeFields, optList, nil)

	// 范围条件的随机组合
	funcRanges := utils.Cartesian(ranges)

	// 串联terms和范围条件组合
	for idx, rq := range funcRanges {
		tq := "	terms := []eq.Map{\n"
		tq += terms + rq
		tq += "	}\n"
		funcRanges[idx] = tq
	}

	return funcRanges
}

// GenTermCond 生成term条件
func GenTermCond(fields []*FieldInfo) string {
	eqm := eqTerms(fields)
	if eqm != "" {
		return "terms := []eq.Map{\n" + eqm + "}\n"
	}
	return ""
}

// GenSortCond 生成sort条件
func GenSortCond(fields []*FieldInfo, topOpt string) string {
	eqs := eqSorts(fields, TopOrder[topOpt]) + "\n"
	return "sorts := " + eqs
}

// WrapTermCond 封装term条件为map
func WrapTermCond(fields string) string {
	if fields != "" {
		return "terms := []eq.Map{\n" + fields + "}\n"
	}
	return ""
}

// GenKnnCond 生成knn条件
func GenKnnCond(fields []*FieldInfo, boolQuery string) string {
	if len(fields) == 0 {
		return ""
	}

	if boolQuery != "" {
		boolQuery = fmt.Sprintf(", eq.WithFilter(%s)", boolQuery)
	}

	f := fields[0]
	kqFmt := `knn := eq.Knn("%s", %s %s)`
	kq := fmt.Sprintf(kqFmt+"\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName), boolQuery)
	return kq
}

// GenAggWithCond 生成多个字段同时聚合
func GenAggWithCond(fields []*FieldInfo, aggFunc string) string {
	aq := ""
	suffix := ""
	prefix := aggFunc
	for idx, f := range fields {
		if idx > 0 {
			prefix = ".With(" + aggFunc
			suffix = ")"
		}
		aq += fmt.Sprintf("%s(\"%s\")%s", prefix, f.EsFieldPath, suffix)
	}
	if len(fields) > 0 {
		return "	aggs :=" + aq + "\n"
	}
	return ""
}

// GenAggWithCondOpt 生成多个字段同时聚合
func GenAggWithCondOpt(fields []*FieldInfo, aggFunc, aggOpt string) string {
	aq := ""
	suffix := ""
	prefix := aggFunc

	// 修正属性参数格式
	if aggOpt != "" && !strings.HasPrefix(aggOpt, ", ") {
		aggOpt = ", " + aggOpt
	}

	for idx, f := range fields {
		if idx > 0 {
			prefix = ".With(" + aggFunc
			suffix = ")"
		}
		aq += fmt.Sprintf("%s(\"%s\"%s)%s", prefix, f.EsFieldPath, aggOpt, suffix)
	}
	if len(fields) > 0 {
		return "	aggs :=" + aq + "\n"
	}
	return ""
}

// GenAggNestedCond 生成多个字段的递归嵌套聚合
func GenAggNestedCond(fields []*FieldInfo, aggFunc string) string {
	// aggs := eq.TermsAgg("zg").Nested(eq.TermsAgg("ak").Nested(eq.TermsAgg("bk")))
	aq := ""
	suffix := ""
	prefix := aggFunc
	for idx, f := range fields {
		if idx > 0 {
			prefix = ".Nested(" + aggFunc
			suffix += ")"
		}
		aq += fmt.Sprintf("%s(\"%s\")", prefix, f.EsFieldPath)
	}
	if len(fields) > 0 {
		return "	aggs :=" + aq + suffix + "\n"
	}
	return ""
}

// AddSubAggCond 对aggs变量添加子聚合条件
func AddSubAggCond(fields []*FieldInfo, aggFunc string) string {
	aq := ""
	prefix := "aggs.Nested(" + aggFunc
	for idx, f := range fields {
		if idx > 0 {
			prefix = ".Nested(" + aggFunc
		}
		aq += fmt.Sprintf("%s(\"%s\"))", prefix, f.EsFieldPath)
	}
	if len(fields) > 0 {
		return "	aggs = " + aq + "\n"
	}
	return ""
}

// GenBoolCond 生成bool条件
func GenBoolCond(mq, tq string, termInShould bool) string {
	if mq == "" && tq == "" {
		return ""
	}

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
func GenESQueryCond(query, agg, sort, size string) string {
	// 组合bool条件
	fq := "	esQuery := &eq.ESQuery{Query: "
	if query != "" {
		fq += query
	}

	if agg != "" {
		fq += ", Agg: aggs"
	}

	if sort != "" {
		fq += ", Sort: sorts"
	}

	if size != "" {
		fq += ", Size: size"
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

// CombineCustom 根据指定列表随机组合数组的元素
// list 字段分组，相当于将宽表拆成多个少字段的表，减少combine组合数
func CombineCustom(items []*FieldInfo, list [][]string, maxCombine int) [][]*FieldInfo {
	var all [][]*FieldInfo
	keyDict := map[string]int{}

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

// term条件列表
func eqTerms(fields []*FieldInfo) string {
	tq := ""
	for _, f := range fields {
		if f.EsFieldType != "text" {
			tq += fmt.Sprintf("		eq.Term(\"%s\", %s),\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName))
		}
	}
	return tq
}

// sort条件列表
func eqSorts(fields []*FieldInfo, order string) string {
	tq := ""
	sf := "eq.Sort"
	for _, f := range fields {
		tq += fmt.Sprintf("%s(\"%s\", \"%s\")", sf, f.EsFieldPath, order)
		sf = ".With"
	}
	return tq
}

// match条件列表
func eqMatches(fields []*FieldInfo) string {
	mq := ""
	for _, f := range fields {
		if f.EsFieldType == "text" {
			mq += fmt.Sprintf("		eq.Match(\"%s\", %s),\n", f.EsFieldPath, utils.ToFirstLower(f.FieldName))
		}
	}
	return mq
}

// range条件
func eqRanges(fields []*FieldInfo, optList [][]string, limitTypes []string) [][]string {
	if len(optList) == 0 {
		optList = CmpOptList
	}
	ranges := [][]string{}
	for _, f := range fields {
		// 如果存在类型限制
		if len(limitTypes) > 0 && !slices.Contains(limitTypes, getTypeMapping(f.EsFieldType)) {
			continue
		}

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
	return ranges
}

// 近期查询条件
func eqRecents(fields []*FieldInfo, rtype string, limitTypes []string) [][]string {
	ranges := [][]string{}
	for _, f := range fields {
		// 如果存在类型限制
		if len(limitTypes) > 0 && !slices.Contains(limitTypes, getTypeMapping(f.EsFieldType)) {
			continue
		}
		tmps := []string{}
		tmp := ""
		gte, gt, lt, lte := "nil", "nil", "nil", "nil"
		gte = utils.ToFirstLower(f.FieldName) + fmt.Sprintf("N%s", rtype)
		gte = fmt.Sprintf("fmt.Sprintf(\"%s\", %s)", RecentFormat[rtype], gte)
		tmp += fmt.Sprintf("		eq.Range(\"%s\", %s, %s, %s, %s),\n", f.EsFieldPath, gte, gt, lt, lte)
		tmps = append(tmps, tmp)
		ranges = append(ranges, tmps)
	}
	return ranges
}

// CurrentFileName 获取当前文件名
func CurrentFileName() string {
	_, file, _, _ := runtime.Caller(1)
	return filepath.Base(file)
}

// CurrentFilePath 获取当前文件路径
func CurrentFilePath() string {
	_, file, _, _ := runtime.Caller(1)
	return file
}
