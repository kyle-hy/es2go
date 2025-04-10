# es2go

## es的mapping生成go结构体struct的工具

根据index的mapping生成结构体，index的注释使用_meta字段的comment存储；字段的注释使用meta字段存储。

## 根据mapping提取的信息生成查询

- [ ] 对text字段做检索（多字段采用合并后的all_text字段来简化）
```json
{
  "match": { "description": "smartphone" }
}

```
- [ ] 对text字段做检索后的命中总数
> 查询响应中的 hits.total.value 字段获取
- [ ] 使用keyword字段随机组合作为过滤条件对text字段做检索
```json
{
  "term": { "category": "electronics" }
}
```
```json
{
  "aggs": {
    "by_category": {
      "terms": { "field": "category" }
    }
  }
}

```
- [ ] 使用keyword字段随机组合作为过滤条件对text字段做检索后的聚合分析
```json
{
  "query": {
    "bool": {
      "filter": [
        { "term": { "category": "electronics" } },
        { "term": { "brand": "apple" } }
      ],
      "must": [
        { "match": { "description": "smartphone" } }
      ]
    }
  },
  "aggs": {
    "top_brands": {
      "terms": {
        "field": "brand"
      }
    }
  }
}

```
- [ ] 使用boolean字段随机组合作为过滤条件对text字段做检索
- [ ] 使用boolean字段随机组合作为过滤条件对text字段做检索后的聚合分析
