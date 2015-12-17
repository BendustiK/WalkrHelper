package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
)

type ConvertedEnergyResponse struct {
	Success       bool `json:"success"`
	CheckedEnergy int  `json:"checked_energy"`
}
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

func main() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true
	proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "multipart/form-data")
		w.Header().Set("Content-Disposition:", "attachment;filename=\""+"shiningbt.mobileconfig\"")
		req.URL, _ = url.Parse("/shiningbt.mobileconfig")
		http.FileServer(http.Dir(".")).ServeHTTP(w, req)
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
	localIp := "127.0.0.1"
	if conn, err := net.Dial("udp", "baidu.com:80"); err == nil {
		defer conn.Close()
		localIp = strings.Split(conn.LocalAddr().String(), ":")[0]
	}

	fmt.Println("你的IP地址是: ", localIp)
	fmt.Println("!!!!!! 第一次使用工具的时候, 请按照[条目0]安装一个描述文件 !!!!!!")
	fmt.Println("0. 在玩儿Walkr的iPad/iPhone上使用Safari打开 [http://" + localIp + ":8888], 会提示下载一个描述文件, 一路安装即可")
	fmt.Println("=========================== 无辜的分割线 ===========================")
	fmt.Println("1. 安装之后在Wifi的代理设置为[手动], 服务器地址为 [" + localIp + "], 端口为 [8888]")
	fmt.Println("2. 打开游戏进入舰桥, 如果能显示能量并且可以领取, 就说明成功")
	fmt.Println("!!!!!! 不用的时候一定关掉[软件]以及[设备上的代理], 否则可能上不了网 !!!!!!")

	log.Fatal(http.ListenAndServe(":8888", proxy))
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
