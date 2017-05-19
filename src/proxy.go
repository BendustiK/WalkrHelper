package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/pborman/uuid"
)

const mobileconfigContent = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadDescription</key>
			<string>Configures restrictions</string>
			<key>PayloadDisplayName</key>
			<string>Restrictions</string>
			<key>PayloadIdentifier</key>
			<string>com.apple.applicationaccess.ABA528C3-F64D-47E2-81BB-6D06DC650E6D</string>
			<key>PayloadType</key>
			<string>com.apple.applicationaccess</string>
			<key>PayloadUUID</key>
			<string>ABA528C3-F64D-47E2-81BB-6D06DC650E6D</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>allowActivityContinuation</key>
			<true/>
			<key>allowAddingGameCenterFriends</key>
			<true/>
			<key>allowAirPlayIncomingRequests</key>
			<true/>
			<key>allowAppCellularDataModification</key>
			<true/>
			<key>allowAppInstallation</key>
			<true/>
			<key>allowAppRemoval</key>
			<true/>
			<key>allowAssistant</key>
			<true/>
			<key>allowAssistantWhileLocked</key>
			<true/>
			<key>allowAutoCorrection</key>
			<true/>
			<key>allowAutomaticAppDownloads</key>
			<true/>
			<key>allowBluetoothModification</key>
			<true/>
			<key>allowBookstore</key>
			<true/>
			<key>allowBookstoreErotica</key>
			<true/>
			<key>allowCamera</key>
			<true/>
			<key>allowChat</key>
			<true/>
			<key>allowCloudBackup</key>
			<true/>
			<key>allowCloudDocumentSync</key>
			<true/>
			<key>allowCloudPhotoLibrary</key>
			<true/>
			<key>allowDefinitionLookup</key>
			<true/>
			<key>allowDeviceNameModification</key>
			<true/>
			<key>allowDictation</key>
			<true/>
			<key>allowEnablingRestrictions</key>
			<true/>
			<key>allowEnterpriseAppTrust</key>
			<true/>
			<key>allowEnterpriseBookBackup</key>
			<true/>
			<key>allowEnterpriseBookMetadataSync</key>
			<true/>
			<key>allowEraseContentAndSettings</key>
			<true/>
			<key>allowExplicitContent</key>
			<true/>
			<key>allowFingerprintForUnlock</key>
			<true/>
			<key>allowFingerprintModification</key>
			<true/>
			<key>allowGameCenter</key>
			<true/>
			<key>allowGlobalBackgroundFetchWhenRoaming</key>
			<true/>
			<key>allowInAppPurchases</key>
			<true/>
			<key>allowKeyboardShortcuts</key>
			<true/>
			<key>allowManagedAppsCloudSync</key>
			<true/>
			<key>allowMultiplayerGaming</key>
			<true/>
			<key>allowMusicService</key>
			<true/>
			<key>allowNews</key>
			<true/>
			<key>allowNotificationsModification</key>
			<true/>
			<key>allowOpenFromManagedToUnmanaged</key>
			<true/>
			<key>allowOpenFromUnmanagedToManaged</key>
			<true/>
			<key>allowPairedWatch</key>
			<true/>
			<key>allowPassbookWhileLocked</key>
			<true/>
			<key>allowPasscodeModification</key>
			<true/>
			<key>allowPhotoStream</key>
			<true/>
			<key>allowPredictiveKeyboard</key>
			<true/>
			<key>allowRadioService</key>
			<true/>
			<key>allowRemoteAppPairing</key>
			<true/>
			<key>allowRemoteScreenObservation</key>
			<true/>
			<key>allowSafari</key>
			<true/>
			<key>allowScreenShot</key>
			<true/>
			<key>allowSharedStream</key>
			<true/>
			<key>allowSpellCheck</key>
			<true/>
			<key>allowSpotlightInternetResults</key>
			<true/>
			<key>allowUIAppInstallation</key>
			<true/>
			<key>allowUIConfigurationProfileInstallation</key>
			<true/>
			<key>allowUntrustedTLSPrompt</key>
			<true/>
			<key>allowVideoConferencing</key>
			<true/>
			<key>allowVoiceDialing</key>
			<true/>
			<key>allowWallpaperModification</key>
			<true/>
			<key>allowiTunes</key>
			<true/>
			<key>forceAirDropUnmanaged</key>
			<false/>
			<key>forceAssistantProfanityFilter</key>
			<false/>
			<key>forceEncryptedBackup</key>
			<false/>
			<key>forceITunesStorePasswordEntry</key>
			<false/>
			<key>forceWatchWristDetection</key>
			<false/>
			<key>forceWiFiWhitelisting</key>
			<false/>
			<key>ratingApps</key>
			<integer>1000</integer>
			<key>ratingMovies</key>
			<integer>1000</integer>
			<key>ratingRegion</key>
			<string>us</string>
			<key>ratingTVShows</key>
			<integer>1000</integer>
			<key>safariAcceptCookies</key>
			<integer>2</integer>
			<key>safariAllowAutoFill</key>
			<true/>
			<key>safariAllowJavaScript</key>
			<true/>
			<key>safariAllowPopups</key>
			<true/>
			<key>safariForceFraudWarning</key>
			<false/>
		</dict>
		<dict>
			<key>PayloadCertificateFileName</key>
			<string>goproxy.github.io.cer</string>
			<key>PayloadContent</key>
			<data>
			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
			uf9Dwz69YA==
			</data>
			<key>PayloadDescription</key>
			<string>Adds a CA root certificate</string>
			<key>PayloadDisplayName</key>
			<string>com.shining.bt</string>
			<key>PayloadIdentifier</key>
			<string>com.apple.security.root.FBB28585-ACC6-4542-8042-014A423660E3</string>
			<key>PayloadType</key>
			<string>com.apple.security.root</string>
			<key>PayloadUUID</key>
			<string>FBB28585-ACC6-4542-8042-014A423660E3</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>WalkrHelper</string>
	<key>PayloadIdentifier</key>
	<string>WalkrHelper.BCE2B675-8973-454E-8346-D6257E7AC2F0</string>
	<key>PayloadRemovalDisallowed</key>
	<false/>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>666BFE3A-0E44-48FB-9ED2-9A464D8026F4</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`

const RemoteAddr = "106.185.47.93"
const PackageFor = "doreamon"
const EpicHack = true
const NeedAuth = false
const Version = "2"

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

	proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*[universe.walkrgame.com|api.walkrhub.com|api.walkrconnect.com|api.walkrorbit.com].*$"))).HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile("^.*[universe.walkrgame.com|api.walkrhub.com|api.walkrconnect.com|api.walkrorbit.com].*$"))).DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
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
	if EpicHack == true {
		proxy.OnResponse(goproxy.UrlMatches(regexp.MustCompile("^.*v1/fleets/(.*)/check_reward$"))).DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
			if body, err := ioutil.ReadAll(resp.Body); err == nil {
				var record CheckEpicRewardResponse
				if err := json.Unmarshal([]byte(body), &record); err == nil {
					record.Data.IsChecked = false
					record.Data.IsFirstTime = false
					// 能量块
					// record.Data.Reward.Type = "cubes"
					// record.Data.Reward.Value = "10000"
					record.Data.Contribution.Value = 30000000
					// DFR
					// record.Data.Reward.Type = "replicator"
					// record.Data.Reward.Value = fmt.Sprintf("map-%v", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(100)+210000)
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
	}

	// 这里应该加一个验证，用来启动或者停止
	authed := false
	if NeedAuth == true {
		currentUUID := strings.Split(uuid.NewUUID().String(), "-")[4]
		md5h := md5.New()
		md5h.Write([]byte(PackageFor + "-" + currentUUID))
		md5str := hex.EncodeToString(md5h.Sum([]byte("")))

		client := &http.Client{}
		v := url.Values{}
		v.Add("u", PackageFor)
		v.Add("id", currentUUID)
		v.Add("md", md5str)
		v.Add("v", Version)

		host := fmt.Sprintf("http://%v:9896/verify?%v", RemoteAddr, v.Encode())
		if req, err := http.NewRequest("GET", host, nil); err != nil {
			fmt.Println(fmt.Sprintf("启动失败: %v", err))
		} else {
			if resp, err := client.Do(req); err != nil {
				fmt.Println(fmt.Sprintf("启动失败: %v", err))
			} else {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil || string(body) != "1" {
					fmt.Println(fmt.Sprintf("启动失败: %v", string(body)))
				} else {
					authed = true
				}
			}
		}
	}
	// 如果需要验证
	if !NeedAuth || authed {
		localIp := "127.0.0.1"
		if conn, err := net.Dial("udp", fmt.Sprintf("%v:9896", RemoteAddr)); err == nil {
			defer conn.Close()
			localIp = strings.Split(conn.LocalAddr().String(), ":")[0]
		} else {

			fmt.Println(err)
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
	} else {
		time.Sleep(5 * time.Second)
	}

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
