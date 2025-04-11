module github.com/kyle-hy/es2go

go 1.24.0

replace github.com/kyle-hy/esquery => /Users/huangzhongfu/Documents/work/gitlab/projects/aibi/esquery

require (
	github.com/elastic/go-elasticsearch/v8 v8.17.1
	github.com/kyle-hy/esquery v1.0.4
	golang.org/x/text v0.23.0
)

require (
	github.com/elastic/elastic-transport-go/v8 v8.6.1 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
)
