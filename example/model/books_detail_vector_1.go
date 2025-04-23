package model

import (
	"github.com/elastic/go-elasticsearch/v8"
	eq "github.com/kyle-hy/esquery"
)

// KnnBooksByAuthor 根据author对v进行向量匹配查找books的详情和总记录数
// author string author
func KnnBooksByAuthor(es *elasticsearch.Client, author string, v []float32) (*eq.Data, *eq.Query, error) {
	filters := []eq.Map{
		eq.Match("author", author),
	}

	knn := eq.Knn("xxx", v, eq.WithFilter(eq.Bool(eq.WithMust(filters))))
	esQuery := &eq.ESQuery{Query: knn}
	return queryBooksList(es, esQuery)
}
