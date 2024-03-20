package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type Place struct {
	Id       int    `json:"ID"`
	Name     string `json:"Name"`
	Address  string `json:"Address"`
	Phone    string `json:"Phone"`
	Location struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
}

type Place_2 struct {
	Id       int    `json:"ID"`
	Name     string `json:"Name"`
	Address  string `json:"Address"`
	Phone    string `json:"Phone"`
	Location struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
}

func createIndex(es *elasticsearch.Client, mapp string) {
	req := esapi.IndicesCreateRequest{
		Index: "places",
		Body:  strings.NewReader(mapp),
	}
	res, _ := req.Do(context.Background(), es)

	defer res.Body.Close()

	fmt.Print(res.StatusCode)
}

func loadData(es *elasticsearch.Client, data []Place) {
	// create bulk indexer
	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:      "places",
		Client:     es,
		NumWorkers: 8,
	})

	if err != nil {
		log.Fatalf("Error while creating bulk indexer")
	}
	ctx := context.Background()

	for _, val := range data {

		valJson, err := json.Marshal(val)
		if err != nil {
			log.Fatalf("failed to convert val (id %d) to json", val.Id)
		}

		err = bulkIndexer.Add(
			ctx,
			esutil.BulkIndexerItem{
				Action:     "index",
				DocumentID: strconv.Itoa(val.Id),
				Body:       bytes.NewReader(valJson),
			})
		if err != nil {
			log.Fatalf("unexpected error: %s", err)
		}
	}

	if err := bulkIndexer.Close(ctx); err != nil {
		log.Fatalf("Unexpected error: %s", err)
	}

	biStats := bulkIndexer.Stats()

	if biStats.NumFailed > 0 {
		fmt.Println("we have ", biStats.NumFailed, " failed")
	} else {
		fmt.Println("data loaded successfully")
		fmt.Println(biStats.NumAdded)
	}

}
func main() {
	mapping := `{
	"mappings": {
	  "properties": {
	      "Id": { "type": "long" },
	      "Name": { "type": "text" },
	      "Address": { "type": "text" },
	      "Phone": { "type": "text" },
	      "Location": { "type": "geo_point" }
	  }
	}
	}`

	cfg := elasticsearch.Config{Addresses: []string{
		"http://localhost:9200",
	}}

	rawData, _ := os.ReadFile("data_2.json")
	var places []Place

	_ = json.Unmarshal(rawData, &places)

	//places_2 := make([]Place_2, len(places))
	//for i, val := range places {
	//	places_2[i].Name = val.Name
	//	places_2[i].Address = val.Address
	//	places_2[i].Location.Lat = val.Location.Lat
	//	places_2[i].Location.Lon = val.Location.Lon
	//	places_2[i].Phone = val.Phone
	//	places_2[i].Id = val.Id
	//}
	//
	//data, _ := json.MarshalIndent(places_2, "", "    ")
	//os.WriteFile("data_2.json", data, 0664)
	es, _ := elasticsearch.NewClient(cfg)
	createIndex(es, mapping)
	loadData(es, places)
}
