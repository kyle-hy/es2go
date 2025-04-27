package generator

// 数值统计聚合
var (
	StatsTypes = []string{"Avg", "Sum", "Min", "Max", "Stats"}
	StatNames  = map[string]string{"Avg": "平均值", "Sum": "总和", "Min": "最小值", "Max": "最大值", "Stats": "统计信息"}
	StatsFuncs = map[string]string{"Avg": AggFuncAvg, "Sum": AggFuncSum, "Min": AggFuncMin, "Max": AggFuncMax, "Stats": AggFuncStats}
)

// 聚合方式枚举
const (
	AggFuncTerms = "eq.TermsAgg" // 分组统计
	AggFuncAvg   = "eq.AvgAgg"   // 计算均值
	AggFuncSum   = "eq.SumAgg"   // 计算总和
	AggFuncMin   = "eq.MinAgg"   // 计算最小值
	AggFuncMax   = "eq.MaxAgg"   // 计算最大值
	AggFuncStats = "eq.StatsAgg" // 计算统计信息
)
