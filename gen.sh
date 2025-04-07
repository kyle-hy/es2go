go run main.go \
    --in example/elasticsearch/cafe-mapping.json \
    --out example/model/cafe.gen.go \
    --struct CafeDocJson \
    --package model \
    --type-mapping example/conf/custom/type-mapping.json \
    --tmpl example/conf/custom/custom-template.tmpl
