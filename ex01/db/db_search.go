package db

import (
	"bytes"
	"context"

	"encoding/json"
	"fmt"
	es "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"io"
	"log"
)

type Place struct {
	Id       int    `json:"ID"`
	Name     string `json:"Name"`
	Address  string `json:"Address"`
	Phone    string `json:"Phone"`
	Location struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"Location"`
}

type SearchRes struct {
	Aggregations struct {
		TypesCount map[string]int `json:"types_count"`
	} `json:"aggregations"`
	Hits Hits `json:"hits"`
}

type Hits struct {
	Total struct {
		Value    int    `json:"value"`
		Relation string `json:"relation"`
	} `json:"total"`
	//MaxScore float64 `json:"max_score"`
	Hit []*Hit `json:"hits"`
}

type Hit struct {
	Source Place `json:"_source"`
}

type PageList struct {
	Name string
	Ref  string
}

type Recommendations struct {
	Name   string  `json:"name"`
	Places []Place `json:"places"`
}

type ApiSearch struct {
	Name     string
	Total    int
	Places   []Place
	PrevPage int
	NextPage int
	LastPage int
}

func (r *ApiSearch) GetPlaces(limit int, offset int) (err error) {
	var res SearchRes
	places, count, err := res.GetPlaces(limit, offset)
	if err != nil {
		return err
	}
	r.Places = places
	r.Total = count
	r.Name = "places"
	return nil
}

func (r *Recommendations) GetPlaces(limit int, offset int, lat, lon string) (err error) {
	client := initESClient()
	ctx := context.Background()

	Body := []byte(`{
		"sort": [
	{"_geo_distance": {
	"Location": {
	"lat": ` + lat + `,
	"lon": ` + lon + `
	},
	"order": "asc",
	"unit": "km",
	"mode": "min",
	"distance_type": "arc",
	"ignore_unmapped": true
	}}]}
`)
	result, _, err := BasicSearch(Body, limit, offset, client, ctx)
	if err != nil {
		return err
	}
	r.Places = result
	r.Name = "Recommendation"
	return nil
}

func initESClient() *es.Client {

	config := es.Config{
		Addresses: []string{"http://localhost:9200"},
	}

	client, err := es.NewClient(config)

	if err != nil {
		log.Fatalln("Cannot create es client")
	}
	return client
}

func (s SearchRes) GetPlaces(limit int, offset int) ([]Place, int, error) {
	client := initESClient()
	ctx := context.Background()

	Body := []byte(`
	{
		"aggs": {
		"types_count": {
			"value_count": {
				"field": "ID"
			}
		}
	},
		"sort": {
		"ID": {
			"order": "asc"
		}
	}
	}`)
	return BasicSearch(Body, limit, offset, client, ctx)
}

func BasicSearch(Body []byte, limit int, offset int, es *es.Client, ctx context.Context) ([]Place, int, error) {
	req := esapi.SearchRequest{
		Index: []string{"places"},
		From:  &offset,
		Size:  &limit,
		Body:  bytes.NewReader(Body),
	}
	res, err := req.Do(ctx, es)
	if err != nil {
		log.Printf("wrong request: %s", err)
	}

	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Fatalln("unknown error")
		}
	}()

	data, err := io.ReadAll(res.Body)
	//fmt.Printf("%s", data)
	r := SearchRes{}

	var foundPlaces []Place

	if err != nil {
		log.Fatalln(err)
	}

	err = json.Unmarshal(data, &r)

	for _, val := range r.Hits.Hit {
		foundPlaces = append(foundPlaces, val.Source)
		fmt.Println("Id: ", val.Source.Id)
	}
	//fmt.Println(r.Aggregations.TypesCount["value"], " total in basic")
	total := r.Aggregations.TypesCount["value"]
	return foundPlaces, total, err
}
