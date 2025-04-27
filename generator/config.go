package generator

import (
	"encoding/json"
	"log"
	"os"
	"strings"
)

// GenConfig 生成器配置
type GenConfig struct {
	MaxCombine   int  `json:"maxCombine"`   // 最大组合的字段数
	TermInShould bool `json:"termInShould"` // 精确条件放在should中可选过滤，评分

	// 数值比较方式的配置
	CmpOptList [][]string `json:"cmpOptList"` // 比较操作列表

	// 超宽表拆分成多个子表的配置
	Combine [][]string `json:"combine"` // 自定义组合的字段列表

	// 字段合并的配置
	AllTextField     string `json:"allTextField"`     // 合并全部文本后的字段名，该字段不作为过滤条件
	AllTextFieldOnly bool   `json:"allTextFieldOnly"` // 合并全部文本后的字段,是否只对该字段做文本检索

	// 范围查询的字段配置
	RangeFields    []string `json:"rangeFields"`    // 做范围查询的字段
	NotRangeFields []string `json:"notRangeFields"` // 不做范围查询的字段

	// 分组统计的字段配置
	TermsFields    []string `json:"termsFields"`    // 做分组统计的字段
	NotTermsFields []string `json:"notTermsFields"` // 不做分组统计的字段

	// 统计信息的字段配置
	StatsFields    []string `json:"statsFields"`    // 做统计的字段
	NotStatsFields []string `json:"notStatsFields"` // 不做统计的字段
}

// getCfgPathByMapping 根据Mapping文件的配置路径获取自定义的配置文件路径
func getCfgPathByMapping(jsonPath string) string {
	indexName := RemoveExt(jsonPath)
	indexName = strings.TrimSuffix(indexName, "_mapping") // 尝试删除索引文件添加的后缀
	indexName = strings.TrimSuffix(indexName, "_Mapping") // 尝试删除索引文件添加的后缀
	indexName = strings.TrimSuffix(indexName, "-mapping") // 尝试删除索引文件添加的后缀
	indexName = strings.TrimSuffix(indexName, "-Mapping") // 尝试删除索引文件添加的后缀
	return indexName + "_custom.json"
}

// LoadCustomGenConfig 加载自定义的生成配置文件
func LoadCustomGenConfig(mappingPath string) *GenConfig {
	cfg := &GenConfig{}
	filePath := getCfgPathByMapping(mappingPath)
	data, err := os.ReadFile(filePath)
	if err == nil {
		log.Printf("custom config: %s", filePath)
		err = json.Unmarshal(data, cfg)
		if err != nil {
			log.Printf("Error unmarshalling JSON from custom generate config %s: %v", filePath, err)
		}
	}

	// 使用全局配置修正缺省配置项
	if cfg.MaxCombine <= 0 {
		cfg.MaxCombine = MaxCombine
	}

	return cfg
}
