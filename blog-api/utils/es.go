package utils

import "github.com/elastic/go-elasticsearch/v7"

var ES *elasticsearch.Client

func InitES() {
	cfg := elasticsearch.Config{
		Addresses: []string{"http://127.0.0.1:9200"},
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	ES = client
}
