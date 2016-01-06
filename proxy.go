package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
)

const mobileconfigContent = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadCertificateFileName</key>
			<string>ca.crt</string>
			<key>PayloadContent</key>
			<data>
			LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUNTakNDQWJX
			Z0F3SUJBZ0lCQURBTEJna3Foa2lHOXcwQkFRVXdTakVqTUNFR0Ex
			VUVDaE1hWjJsMGFIVmkKTG1OdmJTOWxiR0Y2WVhKc0wyZHZjSEp2
			ZUhreEl6QWhCZ05WQkFNVEdtZHBkR2gxWWk1amIyMHZaV3hoZW1G
			eQpiQzluYjNCeWIzaDVNQjRYRFRBd01ERXdNVEF3TURBd01Gb1hE
			VFE1TVRJek1USXpOVGsxT1Zvd1NqRWpNQ0VHCkExVUVDaE1hWjJs
			MGFIVmlMbU52YlM5bGJHRjZZWEpzTDJkdmNISnZlSGt4SXpBaEJn
			TlZCQU1UR21kcGRHaDEKWWk1amIyMHZaV3hoZW1GeWJDOW5iM0J5
			YjNoNU1JR2RNQXNHQ1NxR1NJYjNEUUVCQVFPQmpRQXdnWWtDZ1lF
			QQp2ejlCYkNhSmp4czczVHZjcTNsZVAzMmhBR2VyUTFSZ3ZsWjY4
			WjRuWm1vVkhmbCsyTnIvbTBkbVcrR2RPZnBUCmNzL0t6ZkpqWUdy
			Lzg0eDUyNGZpdVI4R2RaMEhPdFhKenlGNXNlb1duYkJJdXlyMVBi
			RXBnUmhHUU1xcU9VdWoKWUV4ZUxiZk5IUElvSjhYWjFWenl2M1l4
			amJtaldBK1MvdU9lOUhXdERiTUNBd0VBQWFOR01FUXdEZ1lEVlIw
			UApBUUgvQkFRREFnQ2tNQk1HQTFVZEpRUU1NQW9HQ0NzR0FRVUZC
			d01CTUE4R0ExVWRFd0VCL3dRRk1BTUJBZjh3CkRBWURWUjBSQkFV
			d0E0SUJLakFMQmdrcWhraUc5dzBCQVFVRGdZRUFJY0w4aHVTbUdN
			b21wTnVqc3ZlUFRVbk0Kb0VVS3RYNEVoLytzK0RTZlYvVHlJMEkr
			M0dpUHBMcGxFZ0ZXdW9CSUpHaW9zMHIxZEtoNU4wVEdqeFgvUm1H
			bQpxbzdFNGpqSnVvOEdzNVU4L2ZnVGhabXNoYXgybHdMdGJSTndo
			dlVWcjY1R2RhaExzWno4SStoeVNMdWF0VnZSCnFISHEvRlFPUklp
			TnlOcHEvSGc9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
			</data>
			<key>PayloadDescription</key>
			<string>Provides device authentication (certificate or identity).</string>
			<key>PayloadDisplayName</key>
			<string>walkrgame.cert</string>
			<key>PayloadIdentifier</key>
			<string>com.shining.bt.credential</string>
			<key>PayloadOrganization</key>
			<string>com.shining</string>
			<key>PayloadType</key>
			<string>com.apple.security.root</string>
			<key>PayloadUUID</key>
			<string>E92365B3-FE72-4AA6-B23F-401109CD4DFD</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
	</array>
	<key>PayloadDescription</key>
	<string>Cert file for walkr</string>
	<key>PayloadDisplayName</key>
	<string>ShiningBT</string>
	<key>PayloadIdentifier</key>
	<string>com.shining.bt</string>
	<key>PayloadOrganization</key>
	<string>com.shining</string>
	<key>PayloadRemovalDisallowed</key>
	<false/>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>BCE8EE2F-6388-47A2-B029-CAE1FA3355CF</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`

// 领取舰桥能量
type ConvertedEnergyResponse struct {
	Success       bool `json:"success"`
	CheckedEnergy int  `json:"checked_energy"`
}

// 舰桥列表
type PilotListResponse struct {
	Success bool    `json:"success"`
	Pilots  []Pilot `json:"pilots"`
}

type Pilot struct {
	Avatar          string  `json:"avatar"`
	Energy          int     `json:"energy"`
	Id              int     `json:"id"`
	LastConvertedAt int64   `json:"last_converted_at"`
	Name            string  `json:"name"`
	PlanetsCount    int     `json:"planets_count"`
	Position        int     `json:"position"`
	Rsvp            *string `json:"rsvp"`
}

// 传说奖励
type CheckEpicRewardResponse struct {
	Success bool            `json:"success"`
	Data    CheckEpicReward `json:"data"`
}
type CheckEpicReward struct {
	User         User       `json:"user"`
	IsFirstTime  bool       `json:"is_first_time"`
	IsChecked    bool       `json:"is_checked"`
	Reward       CubeReward `json:"reward"`
	Contribution CoinReward `json:"contribution"`
}
type User struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	Level     int    `json:"level"`
	Spaceship string `json:"spaceship"`
}
type CubeReward struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
type CoinReward struct {
	Rate  float64 `json:"rate"`
	Type  string  `json:"type"`
	Value int     `json:"value"`
}

func main() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true
	proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "multipart/form-data")
		w.Header().Set("Content-Disposition:", "attachment;filename=\""+"shiningbt.mobileconfig\"")
		// req.URL, _ = url.Parse("/shiningbt.mobileconfig")
		w.Write([]byte(mobileconfigContent))
		// http.FileServer(http.Dir(".")).ServeHTTP(w, req)
	})

	proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*universe.walkrgame.com.*$"))).HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*universe.walkrgame.com.*$"))).DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		req.Header.Add("Cache-Control", "no-cache,no-store")
		req.Header.Add("Pragma", "no-cache")
		req.Header.Del("If-None-Match")
		return req, nil
	})
	proxy.OnResponse(goproxy.UrlMatches(regexp.MustCompile("^.*v1/pilots?.*$"))).DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if body, err := ioutil.ReadAll(resp.Body); err == nil {
			var record PilotListResponse
			if err := json.Unmarshal([]byte(body), &record); err == nil {
				for index, _ := range record.Pilots {
					record.Pilots[index].Energy = 60000
				}
				dx, _ := json.Marshal(record)
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(dx))
			} else {
				fmt.Println("Unmarshal Err", err)

			}
		} else {
			fmt.Println("Read body Err", err)
		}

		return resp

	})
	proxy.OnResponse(goproxy.UrlMatches(regexp.MustCompile("^.*v1/pilots/(.*)/check$"))).DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		record := &ConvertedEnergyResponse{Success: true, CheckedEnergy: 60000}
		dx, _ := json.Marshal(record)
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(dx))
		return resp

	})

	// https://universe.walkrgame.com/api/v1/fleets/443091/check_reward
	proxy.OnResponse(goproxy.UrlMatches(regexp.MustCompile("^.*v1/fleets/(.*)/check_reward$"))).DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if body, err := ioutil.ReadAll(resp.Body); err == nil {
			var record CheckEpicRewardResponse
			if err := json.Unmarshal([]byte(body), &record); err == nil {
				record.Data.IsChecked = false
				record.Data.IsFirstTime = true
				dx, _ := json.Marshal(record)
				resp.Body = ioutil.NopCloser(bytes.NewBuffer(dx))
			} else {
				fmt.Println("Unmarshal Err", err)

			}
		} else {
			fmt.Println("Read body Err", err)
		}

		return resp

	})

	// TODO: 这里应该加一个验证，用来启动或者停止
	localIp := "127.0.0.1"
	if conn, err := net.Dial("udp", "baidu.com:80"); err == nil {
		defer conn.Close()
		localIp = strings.Split(conn.LocalAddr().String(), ":")[0]
	}
	port := 9897

	fmt.Println("你的IP地址是: ", localIp)
	fmt.Println("!!!!!! 第一次使用工具的时候, 请按照[条目0]安装一个描述文件 !!!!!!")
	fmt.Println("0. 在玩儿Walkr的iPad/iPhone上使用Safari打开 [http://" + localIp + ":" + fmt.Sprintf("%v", port) + "], 会提示下载一个描述文件, 一路安装即可")
	fmt.Println("=========================== 无辜的分割线 ===========================")
	fmt.Println("1. 安装之后在Wifi的代理设置为[手动], 服务器地址为 [" + localIp + "], 端口为 [" + fmt.Sprintf("%v", port) + "]")
	fmt.Println("2. 打开游戏进入舰桥, 如果能显示能量并且可以领取, 就说明成功")
	fmt.Println("!!!!!! 不用的时候一定关掉[软件]以及[设备上的代理], 否则可能上不了网 !!!!!!")

	log.Fatal(http.ListenAndServe(":"+fmt.Sprintf("%v", port), proxy))
}

func UrlMatches(regexps ...*regexp.Regexp) goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		for _, re := range regexps {
			if re.MatchString(req.URL.String()) {
				return true
			}
		}
		return false
	}
}
