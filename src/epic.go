package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"utils"

	"github.com/BurntSushi/toml"
	goerrors "github.com/go-errors/errors"
	goredis "gopkg.in/redis.v2"

	"github.com/op/go-logging"
)

var config PlayerInfos
var leaveComments LeaveComments
var log = logging.MustGetLogger("Walkr")
var format = logging.MustStringFormatter(
	"%{color}%{time:15:04:05.000} %{shortfile} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}",
)

var RoundDuration = 1 * time.Minute
var WaitDuration = 5 * time.Minute
var MaxJoinedTimes = 5
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

// 0. 当前传说任务
type CurrentEpicResponse struct {
	Success bool   `json:"success"`
	FleetId int    `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
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
	Success bool `json:"success"`
}

type PlayerInfo struct {
	Name            string `json:"-"`
	AuthToken       string `json:"auth_token"`
	ClientVersion   string `json:"client_version"`
	Platform        string `json:"platform"`
	Locale          string `json:"locale"`
	Cookie          string `json:"-"`
	ConvertedEnergy int    `json:"-"`
	EpicHelper      bool   `json:"-"`
}

type PlayerInfos struct {
	PlayerInfo []PlayerInfo
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

		// 如果循环开始还有运行的传说，则退出
		_leaveCurrentEpicIfExists(playerInfo)

		// 获取传说列表
		resp, err = _requestEpicList(playerInfo)
		if err != nil {
			log.Error("获取传说列表失败: %v", err)
			_incrRound(playerInfo)
			continue
		}

		invitationEpicIds := _checkInvitationEpics(resp, playerInfo)
		if len(invitationEpicIds) == 0 {
			log.Notice("当前没有邀请的传说, 等待下一次刷新")
			_incrRound(playerInfo)
			continue
		}
		if resp.Body != nil {
			resp.Body.Close()
		}

		// 如果有传说, 随便获取一个传说列表, 找到邀请的传说
		resp, err = _requestFleetList(invitationEpicIds[0], playerInfo)
		if err != nil {
			log.Error("获取舰队列表失败: %v", err)
			_incrRound(playerInfo)
			continue
		}

		fleet := _getInvitationFleet(resp, playerInfo)
		if fleet == nil {
			log.Notice("当前没有邀请的舰队, 等待下次刷新")
			_incrRound(playerInfo)
			continue
		}
		if resp.Body != nil {
			resp.Body.Close()
		}

		appliedOk := _applyInvitedFleet(playerInfo, fleet)
		if appliedOk == false {
			log.Notice("加入舰队[%v:%v]失败, 等待下次刷新", fleet.Name, fleet.Id)
			_incrRound(playerInfo)
			continue
		}

		// BI: 更新加入同一舰队的数量
		_incrJoinedTimes(fleet.Id, playerInfo)

		_leaveComment(playerInfo, fleet, COMMENT_JOINED)

		// 5分钟之后自动退出
		time.Sleep(WaitDuration)

		_leaveComment(playerInfo, fleet, COMMENT_LEAVE)

		if leaveComment := _getRandomComment(); leaveComment != "" {
			_leaveComment(playerInfo, fleet, leaveComment)
		}

		_doLeaveFleet(playerInfo, fleet)

		_incrRound(playerInfo)

	}

}

func _getRandomComment() string {
	commentCountingKey := "epic:comments:counting"
	leaveComment := ""
	if countingMap, err := redis.HGetAllMap(commentCountingKey).Result(); err == nil {
		maxCount := 0
		dataMap := make(map[string]int, 0)
		for comment, countStr := range countingMap {
			count, _ := strconv.Atoi(countStr)
			if count > maxCount {
				maxCount = count
			}
			dataMap[comment] = count
		}
		for comment, count := range dataMap {
			dataMap[comment] = int(math.Pow(float64(maxCount-count), 2)) + 1 // 加1是为了防止大家都平分的情况下导致没有记录选择出来
		}
		leaveComment = utils.GetRandomDataByWeight(dataMap)
		// leaveComment = leaveComments.List[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(leaveComments.List))]

		redis.HIncrBy(commentCountingKey, leaveComment, 1)
	}

	return leaveComment
}

func _requestNewFriendList(playerInfo PlayerInfo) (*http.Response, error) {
	log.Debug("查看是否有好友申请")

	client := &http.Client{}
	v := url.Values{}
	v.Add("platform", playerInfo.Platform)
	v.Add("auth_token", playerInfo.AuthToken)
	v.Add("client_version", playerInfo.ClientVersion)

	host := fmt.Sprintf("https://universe.walkrgame.com/api/v1/users/friend_invitations?%v", v.Encode())

	req, err := utils.GenerateWalkrRequest(host, "GET", playerInfo.Cookie, nil)
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
	req, err := utils.GenerateWalkrRequest(host, "POST", playerInfo.Cookie, bytes.NewBuffer([]byte(b)))
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

func _leaveCurrentEpicIfExists(playerInfo PlayerInfo) bool {
	client := &http.Client{}
	v := url.Values{}
	v.Add("locale", playerInfo.Locale)
	v.Add("platform", playerInfo.Platform)
	v.Add("auth_token", playerInfo.AuthToken)
	v.Add("client_version", playerInfo.ClientVersion)

	host := fmt.Sprintf("https://universe.walkrgame.com/api/v1/fleets/current?%v", v.Encode())
	req, err := utils.GenerateWalkrRequest(host, "GET", playerInfo.Cookie, nil)
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

		var record CurrentEpicResponse
		if err := json.Unmarshal([]byte(body), &record); err != nil {
			log.Error("解析当前舰队信息失败: %v", err)
			return false
		}

		if record.Success == true && record.FleetId != 0 {
			log.Notice("当前有执行中的舰队['%v':%v], 即将离开舰队", record.Name, record.FleetId)

			// 循环开始之前有舰队存在，退出当前舰队
			_doLeaveFleet(playerInfo, &Fleet{Id: record.FleetId, Name: record.Name})
		} else {
			log.Debug("当前没有执行中的舰队, 即将查看邀请列表")
		}

		return true
	} else {
		log.Error("获取当前舰队信息失败: %v", err)
		return false
	}

}

func _requestEpicList(playerInfo PlayerInfo) (*http.Response, error) {
	client := &http.Client{}
	v := url.Values{}
	v.Add("locale", playerInfo.Locale)
	v.Add("platform", playerInfo.Platform)
	v.Add("auth_token", playerInfo.AuthToken)
	v.Add("client_version", playerInfo.ClientVersion)

	host := fmt.Sprintf("https://universe.walkrgame.com/api/v1/epics?%v", v.Encode())
	req, err := utils.GenerateWalkrRequest(host, "GET", playerInfo.Cookie, nil)
	if err != nil {
		return nil, err
	}

	return client.Do(req)

}

func _requestFleetList(invitationEpicId int, playerInfo PlayerInfo) (*http.Response, error) {
	client := &http.Client{}
	v := url.Values{}
	v.Add("locale", playerInfo.Locale)
	v.Add("platform", playerInfo.Platform)
	v.Add("auth_token", playerInfo.AuthToken)
	v.Add("client_version", playerInfo.ClientVersion)
	v.Add("country_code", "US")
	v.Add("epic_id", fmt.Sprintf("%v", invitationEpicId))
	v.Add("limit", "30")
	v.Add("name", "")
	v.Add("offset", "0")

	host := fmt.Sprintf("https://universe.walkrgame.com/api/v1/fleets?%v", v.Encode())
	req, err := utils.GenerateWalkrRequest(host, "GET", playerInfo.Cookie, nil)
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
	req, err := utils.GenerateWalkrRequest(host, "POST", playerInfo.Cookie, bytes.NewBuffer([]byte(b)))
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

		log.Notice("「%v」已经加入舰队[%v:%v], 等待起飞", playerInfo.Name, fleet.Name, fleet.Id)

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
	req, err := utils.GenerateWalkrRequest(host, "POST", playerInfo.Cookie, bytes.NewBuffer([]byte(b)))
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

		log.Notice("「%v」已经留言(%v)", playerInfo.Name, comment)

		return record.Success
	} else {
		log.Error("请求用户留言失败: %v", err)

	}

	return false
}

func _doLeaveFleet(playerInfo PlayerInfo, fleet *Fleet) {
	leaveCount := 1
	for leaveCount <= 5 {
		if leaveOk := _leaveFleet(playerInfo, fleet); leaveOk == true {
			break
		} else {
			log.Error("尝试第%v次离开舰队失败，稍后尝试", leaveCount)
			leaveCount += 1
			time.Sleep(time.Duration(5) * time.Second)
		}
	}
}

func _leaveFleet(playerInfo PlayerInfo, fleet *Fleet) bool {
	client := &http.Client{}

	b, err := json.Marshal(playerInfo)
	if err != nil {
		log.Error("Json Marshal error for %v", err)
		return false
	}

	host := fmt.Sprintf("https://universe.walkrgame.com/api/v1/fleets/%v/leave", fleet.Id)
	req, err := utils.GenerateWalkrRequest(host, "POST", playerInfo.Cookie, bytes.NewBuffer([]byte(b)))
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
			log.Error("「%v」离开舰队失败: %v", playerInfo.Name, err)
			return false
		}

		log.Notice("「%v」退出舰队[%v:%v]成功", playerInfo.Name, fleet.Name, fleet.Id)

		return record.Success
	} else {
		log.Error("「%v」请求离开舰队失败: %v", playerInfo.Name, err)

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

		if epic.InvitationCounts > 0 {
			isInvitation = true
		}
	}

	return isInvitation
}

func _checkInvitationEpics(resp *http.Response, playerInfo PlayerInfo) []int {
	var invitationEpicIds []int

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("读取返回数据失败: %v", err)
		return invitationEpicIds
	}

	var records EpicListResponse
	if err := json.Unmarshal([]byte(body), &records); err != nil {
		log.Error("解析传说列表数据失败: %v", err)
		return invitationEpicIds
	}

	for _, epic := range records.Epics {
		log.Debug("传说[%v], 邀请数量[%v]", epic.Name, epic.InvitationCounts)

		if epic.InvitationCounts > 0 {
			invitationEpicIds = append(invitationEpicIds, epic.Id)
		}
	}

	return invitationEpicIds
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
		log.Debug("%+v", fleet)
		if fleet.IsInvited == true {
			fleet.Quality = _getJoinedTimes(fleet.Id, playerInfo)

			if fleet.Quality <= MaxJoinedTimes {
				fleets = append(fleets, fleet)

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

		return firstFleet
	}

	return nil
}

func (this *PlayerInfo) PlayerId() int {
	playerId, _ := strconv.Atoi(strings.Split(this.AuthToken, ":")[0])
	return playerId
}

// BI相关
func _getRound(playerInfo PlayerInfo) int {
	roundKey := "epic:round"

	currentRound, err := strconv.Atoi(redis.HGet(roundKey, strconv.Itoa(playerInfo.PlayerId())).Val())
	if err != nil || currentRound <= 0 {
		currentRound = 1
	}

	return currentRound

}
func _incrRound(playerInfo PlayerInfo) {
	roundKey := "epic:round"
	redis.HIncrBy(roundKey, strconv.Itoa(playerInfo.PlayerId()), 1)

	time.Sleep(RoundDuration)
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

func _md5String(str string) string {
	md5h := md5.New()
	md5h.Write([]byte(str))
	md5v := md5h.Sum([]byte(""))
	return hex.EncodeToString(md5v)
}

func _saveCommentsToRedis() {
	commentCountingKey := "epic:comments:counting"
	for _, comment := range leaveComments.List {
		redis.HIncrBy(commentCountingKey, comment, 0)
	}
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

	if _, err := toml.DecodeFile("comments.toml", &leaveComments); err != nil {
		log.Error("解析留言列表有问题: %v", err)
		return
	}
	_saveCommentsToRedis()

	// for i := 0; i < 100000; i++ {
	// 	log.Debug("Comment: %v", _getRandomComment())

	// }

	epicHelper := []PlayerInfo{}
	for _, info := range config.PlayerInfo {
		if info.EpicHelper == true {
			epicHelper = append(epicHelper, info)
		}
	}

	if len(epicHelper) == 0 {
		log.Error("没有配置帮飞号信息")
		return
	}

	ch := make(chan int, len(epicHelper))
	for _, playerInfo := range epicHelper {
		go MakeRequest(playerInfo, ch)
	}
	<-ch

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
