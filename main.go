package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/go-redis/redis/v8"
)

var jsonflag = flag.Bool("tojson", false, "get results in json format")

func main() {
	flag.Parse()

	toJson := *jsonflag

	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	var cursor uint64

	routes := make([]RouteInfo, 0)

	for {
		keys, nextCursor, err := rdb.Scan(ctx, cursor, "routeregistry:*", 0).Result()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error scanning keys err:%v\n", err)
			os.Exit(1)
		}

		cursor = nextCursor

		for _, key := range keys {
			parts := strings.Split(key, ":")
			if len(parts) < 3 {
				continue
			}

			// Fetch all fields and values from the hash key
			hashData, err := rdb.HGetAll(ctx, key).Result()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error fetching hash %s for key :%v\n", key, err)
				continue
			}

			routeinfo := hashData["routeinfo"]

			var ri RouteInfo

			err = json.Unmarshal([]byte(routeinfo), &ri)
			if err != nil {
				fmt.Fprintf(os.Stderr, "getting routeinfo %s for key :%v\n", key, err)
				continue
			}

			routes = append(routes, ri)
		}

		if cursor == 0 {
			break
		}
	}

	slices.SortStableFunc(routes, func(a, b RouteInfo) int {
		if a.Resource < b.Resource {
			return -1
		}

		return 1
	})

	if toJson {
		json, err := json.Marshal(routes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to marshal json output:%v\n", err)
			os.Exit(1)
		}

		fmt.Println(string(json))

		return
	}

	for _, route := range routes {
		fmt.Println(route.String())
	}
}

type RouteInfo struct {
	Method        string `json:"method,omitempty"`
	Resource      string `json:"resource,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Subject       string `json:"subject,omitempty"`
	Public        bool   `json:"public,omitempty"`
	Authorization bool   `json:"authorization,omitempty"`
	Type          string `json:"type,omitempty"`
	Timeout       int    `json:"timeout,omitempty"`
}

func (r *RouteInfo) String() string {
	return fmt.Sprintf("resource: %s, method: %s, kind: %s, subject: %v, public: %v, authorization: %v, type: %s, timeout: %d", r.Resource, r.Method, r.Kind, r.Subject, r.Public, r.Authorization, r.Type, r.Timeout)
}
