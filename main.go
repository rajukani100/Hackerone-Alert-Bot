package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type BBP struct {
	Handle          string    `json:"handle"`
	Team_id         int       `json:"team_id"`
	Name            string    `json:"name"`
	Launched_at     time.Time `json:"launched_at"`
	Last_updated_at time.Time `json:"last_updated_at"`
}

func getTotalCount() (int, error) {
	reqBody := `
{
    "operationName": "DiscoveryQuery",
    "variables": {
        "size": 1,
        "from": 0,
        "query": {},
        "filter": {
            "bool": {
                "filter": [
                    {
                        "bool": {
                            "must_not": {
                                "term": {
                                    "team_type": "Engagements::Assessment"
                                }
                            }
                        }
                    },
                    null
                ]
            }
        },
        "sort": [
            {
                "field": "launched_at",
                "direction": "DESC"
            }
        ],
        "post_filters": {
            "my_programs": false,
            "bookmarked": false,
            "campaign_teams": false
        },
        "product_area": "opportunity_discovery",
        "product_feature": "search"
    },
    "query": "query DiscoveryQuery($query: OpportunitiesQuery!, $filter: QueryInput!, $from: Int, $size: Int, $sort: [SortInput!], $post_filters: OpportunitiesFilterInput) {\n  me {\n    id\n    ...OpportunityListMe\n      }\n  opportunities_search(\n    query: $query\n    filter: $filter\n    from: $from\n    size: $size\n    sort: $sort\n    post_filters: $post_filters\n  ) {\n    nodes {\n      ... on OpportunityDocument {\n        id\n        handle\n              }\n      ...OpportunityList\n          }\n    total_count\n      }\n}\n\nfragment OpportunityListMe on User {\n  id\n  ...OpportunityCardMe\n  }\n\nfragment OpportunityCardMe on User {\n  id\n  ...BookmarkMe\n  ...PrivateOpportunitiesMe\n  }\n\nfragment BookmarkMe on User {\n  id\n  }\n\nfragment PrivateOpportunitiesMe on User {\n  id\n  whitelisted_teams {\n    edges {\n      node {\n        id\n        _id\n              }\n          }\n      }\n  }\n\nfragment OpportunityList on OpportunityDocument {\n  id\n  ...OpportunityCard\n  }\n\nfragment OpportunityCard on OpportunityDocument {\n  id\n  team_id\n  name\n  handle\n   launched_at\n  last_updated_at\n  }\n"
}
`

	httpRes, err := http.Post("https://hackerone.com/graphql", "application/json", strings.NewReader(reqBody))
	if err != nil {
		return 0, err
	}
	defer httpRes.Body.Close()

	data, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return 0, err
	}

	result := gjson.Get(string(data), "data.opportunities_search.total_count")
	return int(result.Int()), nil

}

func fetchProgram(from int, wg *sync.WaitGroup, c chan BBP) {
	defer wg.Done()
	reqBody := `
{
    "operationName": "DiscoveryQuery",
    "variables": {
        "size": 24,
        "from": %d,
        "query": {},
        "filter": {
            "bool": {
                "filter": [
                    {
                        "bool": {
                            "must_not": {
                                "term": {
                                    "team_type": "Engagements::Assessment"
                                }
                            }
                        }
                    },
                    null
                ]
            }
        },
        "sort": [
            {
                "field": "launched_at",
                "direction": "DESC"
            }
        ],
        "post_filters": {
            "my_programs": false,
            "bookmarked": false,
            "campaign_teams": false
        },
        "product_area": "opportunity_discovery",
        "product_feature": "search"
    },
    "query": "query DiscoveryQuery($query: OpportunitiesQuery!, $filter: QueryInput!, $from: Int, $size: Int, $sort: [SortInput!], $post_filters: OpportunitiesFilterInput) {\n  me {\n    id\n    ...OpportunityListMe\n      }\n  opportunities_search(\n    query: $query\n    filter: $filter\n    from: $from\n    size: $size\n    sort: $sort\n    post_filters: $post_filters\n  ) {\n    nodes {\n      ... on OpportunityDocument {\n        id\n        handle\n              }\n      ...OpportunityList\n          }\n    total_count\n      }\n}\n\nfragment OpportunityListMe on User {\n  id\n  ...OpportunityCardMe\n  }\n\nfragment OpportunityCardMe on User {\n  id\n  ...BookmarkMe\n  ...PrivateOpportunitiesMe\n  }\n\nfragment BookmarkMe on User {\n  id\n  }\n\nfragment PrivateOpportunitiesMe on User {\n  id\n  whitelisted_teams {\n    edges {\n      node {\n        id\n        _id\n              }\n          }\n      }\n  }\n\nfragment OpportunityList on OpportunityDocument {\n  id\n  ...OpportunityCard\n  }\n\nfragment OpportunityCard on OpportunityDocument {\n  id\n  team_id\n  name\n  handle\n   launched_at\n  last_updated_at\n  }\n"
}
`

	httpRes, err := http.Post("https://hackerone.com/graphql", "application/json", strings.NewReader(fmt.Sprintf(reqBody, from)))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer httpRes.Body.Close()

	data, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return
	}

	result := gjson.Get(string(data), "data.opportunities_search.nodes")
	result.ForEach(func(key, value gjson.Result) bool {
		bbp := BBP{}
		_ = json.Unmarshal([]byte(value.Raw), &bbp)
		c <- bbp
		return true
	})
}

func main() {

	totalCount, err := getTotalCount() // fetch total BBP program count
	if err != nil {
		fmt.Println(err)
		return
	}

	page := math.Ceil(float64(totalCount) / 24) // no of pages

	_ = godotenv.Load(".env")
	URI := os.Getenv("MONGODB_URI")
	client, err := mongo.Connect(options.Client().ApplyURI(URI).SetMaxConnecting(10))
	if err != nil {
		fmt.Println(err)
		return
	}

	collCount, _ := client.Database("hackeroneDB").Collection("BBP").CountDocuments(context.Background(), bson.M{}) // number of document exist in db

	if collCount == 0 {
		bbpChan := make(chan BBP, 10)

		var insertWg sync.WaitGroup
		insertWg.Add(1)
		// save to db
		go func() {
			defer insertWg.Done()
			for bbp := range bbpChan {
				_, err := client.Database("hackeroneDB").Collection("BBP").InsertOne(context.Background(), bbp)
				if err != nil {
					fmt.Println(err)
				}
			}
		}()

		var fetchWg sync.WaitGroup
		for i := 0; i < int(page); i++ {
			fetchWg.Add(1)
			go fetchProgram(i*24, &fetchWg, bbpChan)
		}

		go func() {
			fetchWg.Wait()
			close(bbpChan)
		}()

		insertWg.Wait()
		fmt.Println("Done...")
		return
	}

	updatedBBPchan := make(chan string, 10)
	bbpChan := make(chan BBP, 10)

	var fetchWg sync.WaitGroup
	for i := 0; i < int(page); i++ {
		fetchWg.Add(1)
		go fetchProgram(i*24, &fetchWg, bbpChan)
	}

	go func() {
		fetchWg.Wait()
		close(bbpChan)
	}()

	var insertWg sync.WaitGroup
	insertWg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			for bbp := range bbpChan {
				var tmp BBP // reference to program stored in db
				singleResult := client.Database("hackeroneDB").Collection("BBP").FindOne(context.Background(), bson.D{{Key: "handle", Value: bbp.Handle}})

				if singleResult.Err() == mongo.ErrNoDocuments {
					// program not exist in db, means new program is added
					updatedBBPchan <- bbp.Handle
					_, err := client.Database("hackeroneDB").Collection("BBP").InsertOne(context.Background(), bbp)
					if err != nil {
						fmt.Println(err)
					}
					continue
				}

				err = singleResult.Decode(&tmp)
				if err != nil {
					fmt.Print(err)
					continue
				}

				//compare saved bbp with fetched bbp
				switch tmp.Last_updated_at.Compare(bbp.Last_updated_at) {
				case -1:
					updatedBBPchan <- tmp.Handle

					_, err := client.Database("hackeroneDB").Collection("BBP").ReplaceOne(context.Background(), bson.D{{Key: "handle", Value: tmp.Handle}}, bbp)
					if err != nil {
						fmt.Println(err)
					}

				}
			}
			defer insertWg.Done()
		}()

	}

	insertWg.Wait()
	close(updatedBBPchan)

	//recive all program from channel and put into slice
	var bbpList []string
	var wg sync.WaitGroup
	wg.Add(1)

	go func(wg *sync.WaitGroup) {
		for bbp := range updatedBBPchan {
			fmt.Println("Updated BBP: ", bbp)
			bbpList = append(bbpList, bbp)

		}
		defer wg.Done()
	}(&wg)
	wg.Wait()

	if len(bbpList) > 0 {
		sendEmail(bbpList)
	}
	fmt.Println("Done...")
}
