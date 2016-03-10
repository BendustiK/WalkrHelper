package main

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"github.com/op/go-logging"
	goredis "gopkg.in/redis.v2"
)

var log = logging.MustGetLogger("Walkr")
var format = logging.MustStringFormatter(
	"%{color}%{time:15:04:05.000} %{shortfile} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}",
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

func verifyResponse(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userName := r.FormValue("u")
	uuid := r.FormValue("id")
	md5sum := r.FormValue("md")

	md5h := md5.New()
	md5h.Write([]byte(userName + "-" + uuid))
	md5str := hex.EncodeToString(md5h.Sum([]byte("")))
	log.Debug("请求: 用户[%v], UUID[%v], MD5[%v], 验证MD5[%v]", userName, uuid, md5sum, md5str)
	if md5str != md5sum {
		log.Error("用户[%v]请求MD5认证失败", userName)
		w.Write([]byte("0"))
		return
	}

	if usedMd5, err := redis.HGet("verify:uuid", userName).Result(); err != nil && err.Error() != goredis.Nil.Error() {
		log.Error("用户[%v]请求时Redis错误: %v", err)
		w.Write([]byte("0"))
		return
	} else {
		if usedMd5 == "" {
			redis.HSet("verify:uuid", userName, md5sum)
		} else {
			if usedMd5 != md5sum {
				log.Error("用户[%v]验证已使用过的MD5认证失败", userName)
				w.Write([]byte("0"))
				return
			}
		}
	}

	w.Write([]byte("1"))
	return

}
func main() {
	// 初始化Log
	stdOutput := logging.NewLogBackend(os.Stderr, "", 0)
	stdOutputFormatter := logging.NewBackendFormatter(stdOutput, format)

	logging.SetBackend(stdOutputFormatter)
	redis = goredis.NewClient(redisConf)

	http.HandleFunc("/verify", verifyResponse)
	err := http.ListenAndServe(":9896", nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
