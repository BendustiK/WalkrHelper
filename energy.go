package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	goerrors "github.com/go-errors/errors"

	"github.com/op/go-logging"
)

var RoundDuration = 10 * time.Minute

type PlayerInfo struct {
	Name            string `json:"-"`
	AuthToken       string `json:"auth_token"`
	ClientVersion   string `json:"client_version"`
	Platform        string `json:"platform"`
	Locale          string `json:"-"`
	Cookie          string `json:"-"`
	IfNoneMatch     string `json:"-"`
	ConvertedEnergy int    `json:"converted_energy,string"`
}

type PlayerInfos struct {
	PlayerInfo []PlayerInfo
}

type BoolResponse struct {
	Success bool
}

var round = 1
var config PlayerInfos
var log = logging.MustGetLogger("Walkr")
var format = logging.MustStringFormatter(
	"%{color}%{time:15:04:05.000} %{shortfile} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}",
)

func _generateEnergy() {
	for index, playerInfo := range config.PlayerInfo {
		playerInfo.ConvertedEnergy = rand.New(rand.NewSource(time.Now().UnixNano())).Intn(10000) + 40000
		config.PlayerInfo[index] = playerInfo
	}
}

func MakeRequest() {
	defer func() {
		if r := recover(); r != nil {
			msg := goerrors.Wrap(r, 2).ErrorStack()
			log.Error("程序挂了: %v", msg)
		}
	}()

	log.Notice("===================== 第%v轮 =====================", round)

	_generateEnergy()
	// for key, requestBody := range requestValues {
	for _, playerInfo := range config.PlayerInfo {
		log.Debug("开始刷新「%v」", playerInfo.Name)

		b, err := json.Marshal(playerInfo)
		if err != nil {
			log.Error("转换「%v」的数据格式错误: %v", playerInfo.Name, err)
			continue
		}

		client := &http.Client{}

		host := "https://universe.walkrgame.com/api/v1/pilots/convert"
		req, err := _generateRequest(playerInfo, host, "POST", bytes.NewBuffer([]byte(b)))
		if err != nil {
			log.Error("创建「%v」的请求出错: %v", err)
			continue
		}

		if resp, err := client.Do(req); err == nil {
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Error("读取返回数据失败: %v", err)
				continue
			}

			var record BoolResponse
			if err := json.Unmarshal([]byte(body), &record); err != nil {
				log.Error("「%v」刷新能量失败: %v", playerInfo.Name, err)
				continue
			}

			if record.Success == true {
				log.Notice("「%v」刷新能量成功, 转换能量%v", playerInfo.Name, playerInfo.ConvertedEnergy)
			} else {
				log.Warning("「%v」刷新能量失败, 转换能量%v", playerInfo.Name, playerInfo.ConvertedEnergy)

			}

		}

	}

	round += 1
}

func _generateRequest(playerInfo PlayerInfo, host string, method string, requestBytes *bytes.Buffer) (*http.Request, error) {
	var req *http.Request
	var err error
	if requestBytes == nil {
		req, err = http.NewRequest(method, host, nil)
	} else {
		req, err = http.NewRequest(method, host, requestBytes)
	}
	if err != nil {
		return nil, errors.New("创建Request失败")
	}

	req.Header.Set("Cookie", playerInfo.Cookie)
	if playerInfo.IfNoneMatch != "" {
		req.Header.Add("If-None-Match", playerInfo.IfNoneMatch)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Host", "universe.walkrgame.com")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("User-Agent", "Space Walk/2.1.2 (iPhone; iOS 9.0.2; Scale/2.00)")
	req.Header.Add("Accept-Language", "zh-Hans-CN;q=1")

	return req, nil
}

func main() {
	// 初始化Log
	stdOutput := logging.NewLogBackend(os.Stderr, "", 0)
	stdOutputFormatter := logging.NewBackendFormatter(stdOutput, format)

	logging.SetBackend(stdOutputFormatter)

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

	for true {
		MakeRequest()
		time.Sleep(RoundDuration)
	}

}
