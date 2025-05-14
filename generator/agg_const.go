package generator

// 数值统计聚合
var (
	StatsTypes = []string{"Avg", "Sum", "Min", "Max", "Stats"}
	StatNames  = map[string]string{"Avg": "平均值", "Sum": "总和", "Min": "最小值", "Max": "最大值", "Stats": "统计信息"}
	StatsFuncs = map[string]string{"Avg": AggFuncAvg, "Sum": AggFuncSum, "Min": AggFuncMin, "Max": AggFuncMax, "Stats": AggFuncStats}
)

// 数值极值统计
var (
	TopTypes = []string{"MaxN", "MinN"}
	TopNames = map[string]string{"MaxN": "最大", "MinN": "最小"}
	TopOrder = map[string]string{"MaxN": "desc", "MinN": "asc"}
)

// 近期时间聚合
var (
	RecentTypes  = []string{"Day", "Week", "Month", "Quarter", "Year"}
	RecentNames  = map[string]string{"Day": "为近几天", "Week": "为近几周", "Month": "为近几个月", "Quarter": "为近几个季度", "Year": "为近几年"}
	RecentFormat = map[string]string{"Day": "now-%dd/d", "Week": "now-%dw/w", "Month": "now-%dM/M", "Quarter": "now-%dQ/Q", "Year": "now-%dy/y"}
)

// 日期直方图统计聚合
var (
	DateHistTypes    = []string{"Minute", "Hour", "Day", "Week", "Month", "Quarter", "Year"}
	DateHistNames    = map[string]string{"Minute": "每分钟", "Hour": "每小时", "Day": "每天", "Week": "每周", "Month": "每月", "Quarter": "每季度", "Year": "每年"}
	DateHistInterval = map[string]string{"Minute": "minute", "Hour": "hour", "Day": "day", "Week": "week", "Month": "month", "Quarter": "quarter", "Year": "year"}
)

// 直方图聚合后的数值统计聚合
var (
	HistStatsTypes = []string{"Avg", "Sum", "Min", "Max", "Stats"}
	HistStatNames  = map[string]string{"Avg": "平均值", "Sum": "总和", "Min": "最小值", "Max": "最大值", "Stats": "统计信息"}
	HistStatsFuncs = map[string]string{"Avg": AggFuncAvg, "Sum": AggFuncSum, "Min": AggFuncMin, "Max": AggFuncMax, "Stats": AggFuncStats}
)

// 聚合方式枚举
const (
	AggFuncTerms    = "eq.TermsAgg"         // 分组统计
	AggFuncAvg      = "eq.AvgAgg"           // 计算均值
	AggFuncSum      = "eq.SumAgg"           // 计算总和
	AggFuncMin      = "eq.MinAgg"           // 计算最小值
	AggFuncMax      = "eq.MaxAgg"           // 计算最大值
	AggFuncStats    = "eq.StatsAgg"         // 计算统计信息
	AggFuncHist     = "eq.HistogramAgg"     // 直方图统计
	AggFuncDateHist = "eq.DateHistogramAgg" // 日期直方图统计

	AggOptInterval         = "eq.WithInterval(histInterval)"   // 桶聚合的间隔
	AggOptCalendarInterval = "eq.WithCalendarInterval(\"%s\")" // 桶聚合的时间间隔
)
