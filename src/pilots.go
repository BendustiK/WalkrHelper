package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	goerrors "github.com/go-errors/errors"

	"github.com/op/go-logging"
)

var RoundDuration = 2 * time.Minute
var WaitDuration = 5 * time.Minute
var FleetInvitationCount = make(map[int]int)

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

type BoolResponse struct {
	Success bool
}

type PlayerInfo struct {
	Name            string `json:"-"`
	AuthToken       string `json:"auth_token"`
	ClientVersion   string `json:"client_version"`
	Platform        string `json:"platform"`
	Locale          string `json:"locale"`
	Cookie          string `json:"-"`
	IfNoneMatch     string `json:"-"`
	ConvertedEnergy int    `json:"-"`
}

type PlayerInfos struct {
	PlayerInfo []PlayerInfo
}

var config PlayerInfos
var log = logging.MustGetLogger("Walkr")
var format = logging.MustStringFormatter(
	"%{color}%{time:15:04:05.000} %{shortfile} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}",
)

func MakeRequest() {
	defer func() {
		if r := recover(); r != nil {
			msg := goerrors.Wrap(r, 2).ErrorStack()
			log.Error("程序挂了: %v", msg)
		}
	}()

	// 1. 获取传说列表
	// 2. 获取舰队列表
	// 3. 加入邀请的舰队
	// 4. 留言说明几分钟退出
	// 5. 退出舰队
	currentRound := 1
	for _, playerInfo := range config.PlayerInfo {
		log.Warning("=====================「%v」的第%v次循环 =====================", playerInfo.Name, currentRound)

		// 获取传说列表

		// 每十轮判断是否有好友申请
		_checkFriendInvitation(playerInfo)

	}
	currentRound += 1
}

func _requestNewFriendList(playerInfo PlayerInfo) (*http.Response, error) {
	log.Debug("查看是否有好友申请")

	client := &http.Client{}
	v := url.Values{}
	v.Add("platform", playerInfo.Platform)
	v.Add("auth_token", playerInfo.AuthToken)
	v.Add("client_version", playerInfo.ClientVersion)

	host := fmt.Sprintf("https://api.walkrhub.com/api/v1/users/friend_invitations?%v", v.Encode())

	req, err := _generateRequest(playerInfo, host, "GET", nil)
	if req == nil {
		return nil, err
	}

	return client.Do(req)

}

func _checkFriendInvitation(playerInfo PlayerInfo) bool {
	resp, err := _requestNewFriendList(playerInfo)
	if err != nil {
		return false
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("读取返回数据失败: %v", err)
		return false
	}

	var records NewFriendListResponse
	if err := json.Unmarshal([]byte(body), &records); err != nil {
		log.Error("解析好友列表数据失败: %v", err)
		return false
	}

	if len(records.Data) == 0 {
		log.Debug("没有新的好友申请")
		return false
	}

	for _, friend := range records.Data {
		log.Debug("新的好友申请['%v':%v]", friend.Name, friend.Id)
		if _confirmFriend(playerInfo, friend.Id) == true {
			log.Debug("添加好友['%v':%v]成功", friend.Name, friend.Id)
		} else {
			log.Error("添加好友['%v':%v]失败", friend.Name, friend.Id)
		}
	}

	return true
}

func _confirmFriend(playerInfo PlayerInfo, friendId int) bool {
	client := &http.Client{}

	confirmFriendRequestJson := ConfirmFriendRequest{
		AuthToken:     playerInfo.AuthToken,
		ClientVersion: playerInfo.ClientVersion,
		Platform:      playerInfo.Platform,
		UserId:        friendId,
	}
	b, err := json.Marshal(confirmFriendRequestJson)
	if err != nil {
		log.Error("Json Marshal error for %v", err)
		return false
	}

	host := "https://api.walkrhub.com/api/v1/users/confirm_friend"
	req, err := _generateRequest(playerInfo, host, "POST", bytes.NewBuffer([]byte(b)))
	if err != nil {
		return false
	}

	if resp, err := client.Do(req); err == nil {
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error("读取返回数据失败: %v", err)
			return false
		}

		var record BoolResponse
		if err := json.Unmarshal([]byte(body), &record); err != nil {
			log.Error("通过好友失败: %v", err)
			return false
		}

		return record.Success
	} else {
		log.Error("请求添加用户失败: %v", err)

	}
	return false
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
	req.Header.Add("Host", "api.walkrhub.com")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("User-Agent", "Space Walk/2.1.4 (iPhone; iOS 9.1; Scale/2.00)")
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
