package gocord

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Cluster of Shards connecting to the gateway
type Cluster struct {
	Token       string
	Shards      map[int]*Shard
	TotalShards int
	Dispatch    chan interface{} // chan for gateway dispatch data
	GatewayURL  string
	Options     ClusterOptions
}

// ClusterOptions ...
type ClusterOptions struct {
	Shards      []int // an array of shard IDs
	TotalShards int   // the total shards to spawn
	Presence    Presence
}

func (c *Cluster) fetchRecommendedShards() int {
	req, _ := http.NewRequest(http.MethodGet, RestURL+gatewayPath, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bot %s", c.Token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var decoded gatewayPayload
	decoder.Decode(&decoded)
	c.GatewayURL = decoded.URL

	return decoded.Shards
}

// NewCluster returns a cluster instance
func NewCluster(token string, opts ClusterOptions) *Cluster {
	cluster := &Cluster{
		Token:    token,
		Dispatch: make(chan interface{}),
	}
	cluster.Options = opts
	recShards := cluster.fetchRecommendedShards()

	cluster.Shards = make(map[int]*Shard)
	if len(opts.Shards) == 0 {
		if opts.TotalShards == 0 {
			totalShards := recShards
			cluster.TotalShards = totalShards
			for i := 0; i < recShards; i++ {
				opts.Shards = append(opts.Shards, i)
			}
		}
	} else {
		if opts.TotalShards == 0 {
			cluster.TotalShards = len(opts.Shards)
		}
	}

	for _, id := range opts.Shards {
		shard := NewShard(id, cluster)
		cluster.Shards[id] = shard
	}

	return cluster
}

func (c *Cluster) Spawn() {
	var wg sync.WaitGroup
	for _, shard := range c.Shards {
		wg.Add(1)
		err := shard.Connect()
		if err != nil {
			panic(err)
		}
	}

	wg.Wait()
}