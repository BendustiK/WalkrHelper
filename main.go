package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	goerrors "github.com/go-errors/errors"

	"github.com/op/go-logging"
)

type RequestJson struct {
	AuthToken       string `json:"auth_token"`
	ClientVersion   string `json:"client_version"`
	ConvertedEnergy int    `json:"converted_energy,string"`
	Platform        string `json:"platform"`
	Cookie          string `json:"-"`
	IfNoneMatch     string `json:"-"`
}

type RefreshRecord struct {
	Success bool
}

var round = 1

var requestMapping = map[string]RequestJson{
	"晒宁鼻涕": RequestJson{
		AuthToken:       "370797:brLVsx2TvTbhRyyq65GGEMvqxAvwnWJhrUK4Mu-TJiA",
		ClientVersion:   "2.1.2",
		ConvertedEnergy: 0,
		Platform:        "ios",
		Cookie:          "_spacewalk-server_session=BAh7B0kiD3Nlc3Npb25faWQGOgZFVEkiJTEyNjU1ZTcwM2EyNDc0ZjcyM2EyYmExNDQxNTBiZTIxBjsAVEkiC2xvY2FsZQY7AEZJIgdlbgY7AFQ%3D--e1d43233ff49eb0b4b2e9c319b002cc9254fa958; __cfduid=da3956cb9259cc276c09b648e0b58666a1444714024",
		IfNoneMatch:     "23133a2904c8e76d7b68c8bb3d035c60",
	},
	"豆喵子": RequestJson{
		AuthToken:       "370945:fcAGgjjCzt2pFEBNonnzWnr65fGyNYvCg9SUHFncsDQ",
		ClientVersion:   "2.1.2",
		ConvertedEnergy: 0,
		Platform:        "ios",
		Cookie:          "_spacewalk-server_session=BAh7B0kiD3Nlc3Npb25faWQGOgZFVEkiJTVhZGE2MGMzNzUxY2M0NjAxMmM0YzZkMDJiMTQ0NjYzBjsAVEkiC2xvY2FsZQY7AEZJIgdlbgY7AFQ%3D--71742b31f176efac8ec0110ed3c619fac87f66e9; __cfduid=d534cd265cc169f784f80d48af2bafdf21444184209",
		IfNoneMatch:     "4ecbdbb0c9bfd061cfcbaccaa7a9bffa",
	},
	"宅家豆浆": RequestJson{
		AuthToken:       "473033:C4n9KGgivumMhFNqR29psHFsEZRA7CxxUokosDZExR4",
		ClientVersion:   "2.1.2",
		ConvertedEnergy: 0,
		Platform:        "ios",
		Cookie:          "_spacewalk-server_session=BAh7B0kiD3Nlc3Npb25faWQGOgZFVEkiJTU5YTEwYjk4ZmVlOTc1MDA0ZjFiN2U2Y2YzOWNiNjQzBjsAVEkiC2xvY2FsZQY7AEZJIgp6aC1DTgY7AFQ%3D--e3b0f65033b705e5097f81e595085b636e61f762; __cfduid=db6ae664c9929a7a2c1e400207b4e63c71443003916",
		IfNoneMatch:     "0b88a736758073d10aca6ed18851ba15",
	},
	"宅家白小猫": RequestJson{
		AuthToken:       "477860:afL1_oamjPKvzjVEseBtASF7SwQhBKuEs-U41fyELHz",
		ClientVersion:   "2.1.2",
		ConvertedEnergy: 0,
		Platform:        "ios",
		Cookie:          "_spacewalk-server_session=BAh7B0kiD3Nlc3Npb25faWQGOgZFVEkiJWU4NzdjMGViZTM0YzEwZDkxMDA5ZjYxYmEyYWZhNzVhBjsAVEkiC2xvY2FsZQY7AEZJIgdlbgY7AFQ%3D--45107041393731414997b02f6a158adf5cb13081; __cfduid=d0f1cef52baf17d4647b6a674a0f9df681444279622",
		IfNoneMatch:     "be72b64d7690ddabc24445f689b07d18",
	},
	"宅家飞碟": RequestJson{
		AuthToken:       "543972:u1r8n_TzzmamsuC8AT142-T48sSWrJxo4RYPac-5VXY",
		ClientVersion:   "2.1.2",
		ConvertedEnergy: 0,
		Platform:        "ios",
		Cookie:          "_spacewalk-server_session=BAh7B0kiD3Nlc3Npb25faWQGOgZFVEkiJWM4MjZlY2Q3ZGM1NjcxOWNmZjZhZTAzNTJhY2UxNzVkBjsAVEkiC2xvY2FsZQY7AEZJIgdlbgY7AFQ%3D--ec3366d69081bf6199571e572df8d9c3624e1a8b; __cfduid=d8a93a59248eee274519aa0cd1c90d7e21444280520",
		IfNoneMatch:     "f158c0be4aa2dd39ccee5fb4fc7ecb9d",
	},
	"宅家拉佛": RequestJson{
		AuthToken:       "548754:DTXqwxKJshh_B9xjbXV3RFoxT5TA6rreK3-JBARkxiz",
		ClientVersion:   "2.1.2",
		ConvertedEnergy: 0,
		Platform:        "ios",
		Cookie:          "_spacewalk-server_session=BAh7B0kiD3Nlc3Npb25faWQGOgZFVEkiJWYwZGY2MDNlNmM5ZTdkMjA4OWQyNjk1NTVjZWFhYWU2BjsAVEkiC2xvY2FsZQY7AEZJIgp6aC1DTgY7AFQ%3D--6e63077e6d5dbdf80b2030440f3402ca800300c1; __cfduid=de0eabb3c2a3bd360ac4d168bb59a7f7a1444647143",
		IfNoneMatch:     "f158c0be4aa2dd39ccee5fb4fc7ecb9d",
	},
	"宅家琪萨": RequestJson{
		AuthToken:       "549457:ACQ6b2sKxckyB1PmZdyoishsvVP7VZNoEqgDzuwMUP4",
		ClientVersion:   "2.1.2",
		ConvertedEnergy: 0,
		Platform:        "ios",
		Cookie:          "_spacewalk-server_session=BAh7B0kiD3Nlc3Npb25faWQGOgZFVEkiJTU4ODIxMGExZGExZWMzNDY3MGQwZDM0OTYyY2M4NDUwBjsAVEkiC2xvY2FsZQY7AEZJIgdlbgY7AFQ%3D--919ef308fc20bb139f2ad107545bc768e00937f9; __cfduid=d5a8c09fb0e64413426b1b8fd6ad9e6621444711078",
		IfNoneMatch:     "f158c0be4aa2dd39ccee5fb4fc7ecb9d",
	},
}

var recordMapping = make(map[uint64]string)
var log = logging.MustGetLogger("Xueqiu")
var format = logging.MustStringFormatter(
	"%{color}%{time:15:04:05.000} %{shortfile} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}",
)

func _generateEnergy() map[string]RequestJson {
	refreshedValus := requestMapping
	for key, value := range refreshedValus {
		value.ConvertedEnergy = rand.New(rand.NewSource(time.Now().UnixNano())).Intn(10000) + 40000
		refreshedValus[key] = value
	}

	return refreshedValus
}

func MakeRequest() {
	defer func() {
		if r := recover(); r != nil {
			msg := goerrors.Wrap(r, 2).ErrorStack()
			log.Error("Panic recovered from Make Request", msg)
		}
	}()

	log.Notice("==================== Round %v ====================", round)

	requestValues := _generateEnergy()
	for key, requestBody := range requestValues {
		log.Debug("Start to make request for %v", key)
		b, err := json.Marshal(requestBody)
		if err != nil {
			log.Error("Json Marshal error for %v: %v", key, err)
		}

		client := &http.Client{}
		if req, err := http.NewRequest("POST", "https://universe.walkrgame.com/api/v1/pilots/convert", bytes.NewBuffer([]byte(b))); err == nil {

			req.Header.Set("Cookie", requestBody.Cookie)
			if requestBody.IfNoneMatch != "" {
				req.Header.Add("If-None-Match", requestBody.IfNoneMatch)
			}
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Host", "universe.walkrgame.com")
			req.Header.Add("Accept", "*/*")
			req.Header.Add("User-Agent", "Space Walk/2.1.2 (iPhone; iOS 9.0.2; Scale/2.00)")
			req.Header.Add("Accept-Language", "zh-Hans-CN;q=1, en-CN;q=0.9")

			if resp, err := client.Do(req); err == nil {
				defer resp.Body.Close()

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Error("Reading error: %v", err)
				}

				var records RefreshRecord
				if err := json.Unmarshal([]byte(body), &records); err != nil {
					log.Error("Unmarshal failed: %v", err)
				}

				log.Notice("Refresh Result: %v, Converted Energy: %v", records.Success, requestBody.ConvertedEnergy)

			} else {
				log.Error("Do Request Error: %v", err)

			}

		} else {
			log.Error("Init request failed: %v", err)

		}

	}

	round += 1
}

func main() {
	stdOutput := logging.NewLogBackend(os.Stderr, "", 0)
	stdOutputFormatter := logging.NewBackendFormatter(stdOutput, format)

	logging.SetBackend(stdOutputFormatter)

	for true {
		MakeRequest()
		// randSeconds := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(60)
		time.Sleep((15 * time.Minute))
	}

}
