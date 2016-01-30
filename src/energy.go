package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"utils"

	"github.com/BurntSushi/toml"
	goerrors "github.com/go-errors/errors"
	goredis "gopkg.in/redis.v2"

	"github.com/op/go-logging"
)

var RoundDuration = 10 * time.Minute
var config PlayerInfos
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

type PlayerInfo struct {
	Name            string `json:"-"`
	AuthToken       string `json:"auth_token"`
	ClientVersion   string `json:"client_version"`
	Platform        string `json:"platform"`
	Locale          string `json:"-"`
	Cookie          string `json:"-"`
	ConvertedEnergy int    `json:"converted_energy,string"`
	EpicHelper      bool   `json:"-"`
}

type PlayerInfos struct {
	PlayerInfo []PlayerInfo
}

type BoolResponse struct {
	Success bool
}

type ConfirmFriendRequest struct {
	AuthToken     string `json:"auth_token"`
	UserId        int    `json:"user_id"`
	ClientVersion string `json:"client_version"`
	Platform      string `json:"platform"`
}

type NewFriendListResponse struct {
	Data []Friend `json:"data"`
}
type Friend struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func MakeRequest(playerInfo PlayerInfo, ch chan int) {
	for {
		select {
		case <-ch:
			fmt.Println("exiting...")
			ch <- 1
			break
		default:
		}

		defer func() {
			if r := recover(); r != nil {
				msg := goerrors.Wrap(r, 2).ErrorStack()
				log.Error("程序挂了: %v", msg)
				return
			}
		}()
		// log.Notice("===================== 第%v轮 =====================", curRound)
		// _checkFriendInvitation(playerInfo)

		_convertEnegeryToPilots(playerInfo)

		_incrRound(playerInfo)
		time.Sleep(RoundDuration)
	}

}
func _convertEnegeryToPilots(playerInfo PlayerInfo) bool {
	playerInfo = _generateEnergy(playerInfo)

	b, err := json.Marshal(playerInfo)
	if err != nil {
		log.Error("转换「%v」的数据格式错误: %v", playerInfo.Name, err)
		return false
	}

	client := &http.Client{}

	host := "https://api.walkrhub.com/api/v1/pilots/convert"
	req, err := utils.GenerateWalkrRequest(host, "POST", playerInfo.Cookie, bytes.NewBuffer([]byte(b)))
	if err != nil {
		log.Error("创建「%v」的请求出错: %v", err)
		return false

	}

	if resp, err := client.Do(req); err == nil {

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error("读取返回数据失败: %v", err)
			return false
		}

		var record BoolResponse
		if err := json.Unmarshal([]byte(body), &record); err != nil {
			log.Error("「%v」刷新能量失败: %v", playerInfo.Name, err)
			return false
		}

		if record.Success == true {
			log.Notice("第%v轮「%v」刷新能量成功, 转换能量%v", _getRound(playerInfo), playerInfo.Name, playerInfo.ConvertedEnergy)
		} else {
			log.Warning("「%v」刷新能量失败, 转换能量%v", playerInfo.Name, playerInfo.ConvertedEnergy)

		}

		resp.Body.Close()
		return true

	} else {
		log.Error("创建请求失败: %v", err)
		return false

	}
}

func _generateEnergy(playerInfo PlayerInfo) PlayerInfo {
	playerInfo.ConvertedEnergy = rand.New(rand.NewSource(time.Now().UnixNano())).Intn(10000) + 50000
	return playerInfo
}

// BI相关
func _getRound(playerInfo PlayerInfo) int {
	roundKey := "energy:round"

	currentRound, err := strconv.Atoi(redis.HGet(roundKey, strconv.Itoa(playerInfo.PlayerId())).Val())
	if err != nil || currentRound <= 0 {
		currentRound = 1
	}

	return currentRound
}
func _incrRound(playerInfo PlayerInfo) {
	roundKey := "energy:round"
	redis.HIncrBy(roundKey, playerInfo.PlayerId(), 1)

}

func (this *PlayerInfo) PlayerId() int {
	playerId, _ := strconv.Atoi(strings.Split(this.AuthToken, ":")[0])
	return playerId
}

func main() {
	ch := make(chan int, 10)

	// 初始化Log
	stdOutput := logging.NewLogBackend(os.Stderr, "", 0)
	stdOutputFormatter := logging.NewBackendFormatter(stdOutput, format)

	logging.SetBackend(stdOutputFormatter)

	redis = goredis.NewClient(redisConf)

	// 读取参数来获得配置文件的名称
	argCount := len(os.Args)
	if argCount == 0 {
		log.Warning("需要输入配置文件名称: 格式 '-c fileName'")
		return
	}

	cmd := flag.String("c", "help", "配置文件名称")
	flag.Parse()
	if *cmd == "help" {
		log.Warning("需要输入配置文件名称: 格式 '-c fileName'")
		return
	}

	if _, err := toml.DecodeFile(*cmd, &config); err != nil {
		log.Error("配置文件有问题: %v", err)
		return
	}
	for _, playerInfo := range config.PlayerInfo {
		go MakeRequest(playerInfo, ch)
	}
	<-ch

}
