package main

import (
	"fmt"
	"strings"
	"time"

	goredis "gopkg.in/redis.v2"
)

var redis *goredis.Client

var redisConf = &goredis.Options{
	Network:      "tcp",
	Addr:         "localhost:6379",
	Password:     "",
	DB:           0,
	DialTimeout:  5 * time.Second,
	ReadTimeout:  5 * time.Second,
	WriteTimeout: 5 * time.Second,
	PoolSize:     20,
	IdleTimeout:  60 * time.Second,
}

func main() {

	redis = goredis.NewClient(redisConf)

	for _, key := range redis.Keys("energy:*:round").Val() {
		fmt.Println(key)
		playerId := strings.Split(key, ":")[1]
		cnt, _ := redis.Get(key).Int64()
		redis.HIncrBy("energy:round", playerId, cnt)

	}

	for _, key := range redis.Keys("epic:*:round").Val() {
		fmt.Println(key)
		playerId := strings.Split(key, ":")[1]
		cnt, _ := redis.Get(key).Int64()
		redis.HIncrBy("epic:round", playerId, cnt)

	}

}
