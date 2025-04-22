package generator

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/kyle-hy/es2go/utils"
)

// GenConfig 生成器配置
type GenConfig struct {
	Combine      [][]string `json:"combine"`      // 自定义组合的字段列表
	MaxCombine   int        `json:"maxCombine"`   // 最大组合的字段数
	TermInShould bool       `json:"termInShould"` // 精确条件放在should中可选过滤，评分
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
	if err != nil {
		return cfg
	}

	err = json.Unmarshal(data, cfg)
	if err != nil {
		log.Printf("Error unmarshalling JSON from custom generate config %s: %v", filePath, err)
		return cfg
	}

	log.Printf("custom config: %s", filePath)

	utils.JPrint(cfg)
	return cfg
}
