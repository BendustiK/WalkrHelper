package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	goerrors "github.com/go-errors/errors"
	goredis "gopkg.in/redis.v2"

	"github.com/op/go-logging"
)

var RoundDuration = 2 * time.Minute
var WaitDuration = 5 * time.Minute
var MAX_JOIN_TIMES = 5
var FleetInvitationCount = make(map[int]int)
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

const (
	COMMENT_JOINED = "我进来啦，我会在五分钟之后自动退队。如果退队的时候还没有捐献完毕，不要着急，重新邀请就好。不过请记住，同一舰队邀请数量达到五次，我会忽略邀请的。谢谢!"
	COMMENT_LEAVE  = "关于离开舰队, 大家有话说."
)

type LeaveComments struct {
	List []string
}

type CommentRequest struct {
	AuthToken     string `json:"auth_token"`
	ClientVersion string `json:"client_version"`
	Platform      string `json:"platform"`
	Locale        string `json:"locale"`
	Text          string `json:"text"`
}

type ConfirmFriendRequest struct {
	AuthToken     string `json:"auth_token"`
	UserId        int    `json:"user_id"`
	ClientVersion string `json:"client_version"`
	Platform      string `json:"platform"`
}

// 1. 传说列表Resp
type EpicListResponse struct {
	Epics []Epic `json:"epics"`
}
type Epic struct {
	Id               int    `json:"id"`
	Name             string `json:"name"`
	InvitationCounts int    `json:"invitation_counts"`
}

// 2. 飞传说中的舰队列表Resp
type FleetListResponse struct {
	Fleets []Fleet `json:"fleets"`
}
type Fleet struct {
	Id        int     `json:"id"`
	Name      string  `json:"name"`
	IsInvited bool    `json:"is_invited"`
	Captain   Captain `json:"captain"`
	Quality   int
}
type Captain struct {
	Name string `json:"name"`
}

// 3. 舰队详细信息
type FleetDetailInfo struct {
	Id      int      `json:"id"`
	Name    string   `json:"name"`
	EpicId  int      `json:"epic_id"`
	Members []Member `json:"members"`
}
type Member struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// 4. 好友申请
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

func MakeRequest(playerInfo PlayerInfo) {
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
	// for _, playerInfo := range config.PlayerInfo {
	currentRound := _getRound(playerInfo)
	log.Warning("=====================「%v」的第%v次循环 =====================", playerInfo.Name, currentRound)

	// 获取传说列表
	var resp *http.Response
	var err error

	// 每十轮判断是否有好友申请
	if currentRound%2 == 0 {
		_checkFriendInvitation(playerInfo)
	}

	// 获取传说列表
	resp, err = _requestEpicList(playerInfo)
	if err != nil {
		log.Error("获取传说列表失败: %v", err)
		continue
	}

	hasInvitation := _checkInvitationCount(resp, playerInfo)
	if hasInvitation == false {
		log.Notice("当前没有邀请的传说, 等待下一次刷新")
		continue
	}
	if resp.Body != nil {
		resp.Body.Close()
	}

	// 如果有传说, 随便获取一个传说列表, 找到邀请的传说
	resp, err = _requestFleetList(playerInfo)
	if err != nil {
		log.Error("获取舰队列表失败: %v", err)
		continue
	}

	fleet := _getInvitationFleet(resp, playerInfo)
	if fleet == nil {
		log.Notice("当前没有邀请的舰队, 等待下次刷新")
		continue
	}
	if resp.Body != nil {
		resp.Body.Close()
	}

	appliedOk := _applyInvitedFleet(playerInfo, fleet)
	if appliedOk == false {
		log.Notice("加入舰队[%v:%v]失败, 等待下次刷新", fleet.Name, fleet.Id)
		continue
	}

	// BI: 更新加入同一舰队的数量
	_incrJoinedTimes(fleet.Id, playerInfo)

	_leaveComment(playerInfo, fleet, COMMENT_JOINED)

	// 5分钟之后自动退出
	time.Sleep(WaitDuration)

	_leaveComment(playerInfo, fleet, COMMENT_LEAVE)

	var leaveComments LeaveComments
	if _, err := toml.DecodeFile("comments.toml", &leaveComments); err != nil {
		log.Error("解析留言列表有问题: %v", err)
	} else {
		leaveComment := leaveComments.List[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(leaveComments.List))]
		_leaveComment(playerInfo, fleet, leaveComment)
	}

	leaveCount := 0
	for leaveCount < 5 {
		if leaveOk := _leaveFleet(playerInfo, fleet); leaveOk == true {
			break
		} else {
			log.Error("尝试第%v次离开舰队失败，稍后尝试", leaveCount)
			leaveCount += 1
			time.Sleep(time.Duration(5) * time.Second)
		}
	}

	// }
	_incrRound(playerInfo)

}

func _requestNewFriendList(playerInfo PlayerInfo) (*http.Response, error) {
	log.Debug("查看是否有好友申请")

	client := &http.Client{}
	v := url.Values{}
	v.Add("platform", playerInfo.Platform)
	v.Add("auth_token", playerInfo.AuthToken)
	v.Add("client_version", playerInfo.ClientVersion)

	host := fmt.Sprintf("https://universe.walkrgame.com/api/v1/users/friend_invitations?%v", v.Encode())

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

	host := "https://universe.walkrgame.com/api/v1/users/confirm_friend"
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

func _requestEpicList(playerInfo PlayerInfo) (*http.Response, error) {
	client := &http.Client{}
	v := url.Values{}
	v.Add("locale", playerInfo.Locale)
	v.Add("platform", playerInfo.Platform)
	v.Add("auth_token", playerInfo.AuthToken)
	v.Add("client_version", playerInfo.ClientVersion)

	host := fmt.Sprintf("https://universe.walkrgame.com/api/v1/epics?%v", v.Encode())
	req, err := _generateRequest(playerInfo, host, "GET", nil)
	if err != nil {
		return nil, err
	}

	return client.Do(req)

}

func _requestFleetList(playerInfo PlayerInfo) (*http.Response, error) {
	client := &http.Client{}
	v := url.Values{}
	v.Add("locale", playerInfo.Locale)
	v.Add("platform", playerInfo.Platform)
	v.Add("auth_token", playerInfo.AuthToken)
	v.Add("client_version", playerInfo.ClientVersion)
	v.Add("country_code", "US")
	v.Add("epic_id", "14")
	v.Add("limit", "30")
	v.Add("name", "")
	v.Add("offset", "0")

	host := fmt.Sprintf("https://universe.walkrgame.com/api/v1/fleets?%v", v.Encode())
	req, err := _generateRequest(playerInfo, host, "GET", nil)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

func _applyInvitedFleet(playerInfo PlayerInfo, fleet *Fleet) bool {
	client := &http.Client{}
	b, err := json.Marshal(playerInfo)
	if err != nil {
		log.Error("Json Marshal error for %v", err)
		return false
	}

	host := fmt.Sprintf("https://universe.walkrgame.com/api/v1/fleets/%v/apply", fleet.Id)
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
			log.Error("加入舰队失败: %v", err)
			return false
		}

		log.Notice("已经加入舰队[%v:%v], 等待起飞", fleet.Name, fleet.Id)

		// BI: 加入的舰队信息
		_setFleetTime(fleet, "joinedTime", playerInfo)

		// BI: Round信息
		redis.HMSet(_roundInfoKey(playerInfo), "joinedFleetId", fmt.Sprintf("%v", fleet.Id), "joinedTime", fmt.Sprintf("%v", time.Now().UTC().Unix()))

		return record.Success
	} else {
		log.Error("请求加入舰队失败: %v", err)

	}

	return false
}

func _leaveComment(playerInfo PlayerInfo, fleet *Fleet, comment string) bool {
	client := &http.Client{}

	commentRequestJson := CommentRequest{
		AuthToken:     playerInfo.AuthToken,
		ClientVersion: playerInfo.ClientVersion,
		Platform:      playerInfo.Platform,
		Locale:        playerInfo.Locale,
		Text:          comment,
	}
	b, err := json.Marshal(commentRequestJson)
	if err != nil {
		log.Error("Json Marshal error for %v", err)
		return false
	}

	host := fmt.Sprintf("https://universe.walkrgame.com/api/v1/fleets/%v/comment", fleet.Id)
	req, err := _generateRequest(playerInfo, host, "POST", bytes.NewBuffer([]byte(b)))
	if err != nil {
		log.Error("请求留言失败 %v", err)
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
			log.Error("留言失败: %v", err)
			return false
		}

		log.Notice("已经留言(%v)", comment)

		_saveComment(fleet, comment, playerInfo)
		return record.Success
	} else {
		log.Error("请求用户留言失败: %v", err)

	}

	return false
}

func _leaveFleet(playerInfo PlayerInfo, fleet *Fleet) bool {
	client := &http.Client{}

	b, err := json.Marshal(playerInfo)
	if err != nil {
		log.Error("Json Marshal error for %v", err)
		return false
	}

	host := fmt.Sprintf("https://universe.walkrgame.com/api/v1/fleets/%v/leave", fleet.Id)
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
			log.Error("离开舰队失败: %v", err)
			return false
		}

		log.Notice("退出舰队[%v:%v]成功", fleet.Name, fleet.Id)

		// BI: 为舰队设置离开标志
		_setFleetTime(fleet, "leaveTime", playerInfo)
		// BI: Round信息
		redis.HMSet(_roundInfoKey(playerInfo), "leaveTime", fmt.Sprintf("%v", time.Now().UTC().Unix()))

		return record.Success
	} else {
		log.Error("请求离开舰队失败: %v", err)

	}

	return false
}

func _checkInvitationCount(resp *http.Response, playerInfo PlayerInfo) bool {
	isInvitation := false

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("读取返回数据失败: %v", err)
		return isInvitation
	}

	var records EpicListResponse
	if err := json.Unmarshal([]byte(body), &records); err != nil {
		log.Error("解析传说列表数据失败: %v", err)
		return isInvitation
	}

	for _, epic := range records.Epics {
		log.Debug("传说[%v], 邀请数量[%v]", epic.Name, epic.InvitationCounts)

		// BI: Round信息
		redis.HIncrBy(_roundInfoKey(playerInfo), "initationCount", int64(epic.InvitationCounts))

		if epic.InvitationCounts > 0 {
			isInvitation = true
		}
	}

	return isInvitation
}

func _getInvitationFleet(resp *http.Response, playerInfo PlayerInfo) *Fleet {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("读取返回数据失败: %v", err)
		return nil
	}

	var records FleetListResponse
	if err := json.Unmarshal([]byte(body), &records); err != nil {
		log.Error("解析传说列表数据失败: %v", err)
		return nil
	}

	var fleets Fleets
	for _, fleet := range records.Fleets {
		if fleet.IsInvited == true {
			fleet.Quality = _getJoinedTimes(fleet.Id, playerInfo)

			if fleet.Quality <= MAX_JOIN_TIMES {
				fleets = append(fleets, fleet)

				// BI: Round信息
				redis.HIncrBy(_roundInfoKey(playerInfo), "validInvitationCount", 1)

			} else {
				log.Error("舰队[%v:%v] by (%v): 已经到达自动帮飞次数上限, 加入黑名单", fleet.Name, fleet.Id, fleet.Captain.Name)

			}

		}
	}

	if len(fleets) > 0 {
		// 加入次数少的队伍优先进入, 防止恶意邀请阻塞进程
		sort.Sort(fleets)

		firstFleet := &fleets[0]
		log.Notice("舰队[%v:%v] by (%v): 正在邀请, 优先度(%v)", firstFleet.Name, firstFleet.Id, firstFleet.Captain.Name, firstFleet.Quality)

		// BI: 设置邀请的舰队信息
		_saveFleetInfo(firstFleet, playerInfo)

		// BI: Round信息
		redis.HSet(_roundInfoKey(playerInfo), "chosenFleetId", fmt.Sprintf("%v", firstFleet.Id))

		return firstFleet
	}

	return nil
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
	req.Header.Add("User-Agent", "Space Walk/2.1.4 (iPhone; iOS 9.1; Scale/2.00)")
	req.Header.Add("Accept-Language", "zh-Hans-CN;q=1")

	return req, nil
}

func (this *PlayerInfo) PlayerId() int {
	playerId, _ := strconv.Atoi(strings.Split(this.AuthToken, ":")[0])
	return playerId
}

// BI相关
func _getRound(playerInfo PlayerInfo) int {
	currentRound, err := strconv.Atoi(redis.Get(fmt.Sprintf("epic:%v:round", playerInfo.PlayerId())).Val())
	if err != nil || currentRound <= 0 {
		currentRound = 1
	}

	return currentRound
}
func _incrRound(playerInfo PlayerInfo) {
	redis.Incr(fmt.Sprintf("epic:%v:round", playerInfo.PlayerId()))
}

func _getJoinedTimes(fleetId int, playerInfo PlayerInfo) int {
	times, err := strconv.Atoi(redis.HGet(fmt.Sprintf("epic:%v:fleet:times", playerInfo.PlayerId()), fmt.Sprintf("%v", fleetId)).Val())
	if err != nil || times <= 0 {
		times = 0
	}

	return times
}

func _incrJoinedTimes(fleetId int, playerInfo PlayerInfo) {
	redis.HIncrBy(fmt.Sprintf("epic:%v:fleet:times", playerInfo.PlayerId()), fmt.Sprintf("%v", fleetId), 1)
}

func _roundInfoKey(playerInfo PlayerInfo) string {
	return fmt.Sprintf("epic:%v:round:%v:info", playerInfo.PlayerId(), _getRound(playerInfo))
}

func _saveFleetInfo(fleet *Fleet, playerInfo PlayerInfo) {
	fleetKey := fmt.Sprintf("epic:%v:fleet:%v:info", playerInfo.PlayerId(), fleet.Id)
	redis.HMSet(fleetKey, "id", fmt.Sprintf("%v", fleet.Id), "fleetName", fleet.Name, "captainName", fleet.Captain.Name, "quality", fmt.Sprintf("%v", fleet.Quality), "round", fmt.Sprintf("%v", _getRound(playerInfo)), "invitedTime", fmt.Sprintf("%v", time.Now().UTC().Unix()), "joinedTime", "0", "leaveTime", "0")
}
func _setFleetTime(fleet *Fleet, field string, playerInfo PlayerInfo) {
	fleetKey := fmt.Sprintf("epic:%v:fleet:%v:info", playerInfo.PlayerId(), fleet.Id)
	redis.HSet(fleetKey, field, fmt.Sprintf("%v", time.Now().UTC().Unix()))
}

func _saveComment(fleet *Fleet, comment string, playerInfo PlayerInfo) {
	fleetKey := fmt.Sprintf("epic:%v:fleet:%v:comments", playerInfo.PlayerId(), fleet.Id)
	redis.LPush(fleetKey, comment)
}

func main() {
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

	for true {

		for _, playerInfo := range config.PlayerInfo {
			MakeRequest(playerInfo)
			time.Sleep(RoundDuration)
		}

	}

}

type Fleets []Fleet

func (ms Fleets) Len() int {
	return len(ms)
}

func (ms Fleets) Less(i, j int) bool {
	return ms[i].Quality < ms[j].Quality
}

func (ms Fleets) Swap(i, j int) {
	ms[i], ms[j] = ms[j], ms[i]
}
