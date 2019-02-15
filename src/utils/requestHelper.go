package utils

import (
	"bytes"
	"errors"
	"net/http"
)

func GenerateWalkrRequest(host string, method string, cookie string, requestBytes *bytes.Buffer) (*http.Request, error) {
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

	req.Header.Set("Cookie", cookie)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Host", "universe.walkrgame.com")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("User-Agent", "Walkr/4.8.4 (iPhone; iOS 12.1.3; Scale/2.00)")
	req.Header.Add("Accept-Language", "zh-Hans-CN;q=1, en-CN;q=0.9")

	return req, nil
}
