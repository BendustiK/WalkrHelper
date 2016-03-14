#!/bin/bash -e

# echo "正在生成64位的Proxy"
# GOOS=windows GOARCH=amd64 go build  -o proxy64.exe proxy.go

echo "正在生成32位的Proxy"
GOOS=windows GOARCH=386 go build -ldflags "-s -w"  -o proxy32.exe src/proxy.go
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w"  -o proxy src/proxy.go
if which upx 2>/dev/null; then
  echo "正在使用UPX压缩"
	upx proxy32.exe
	upx proxy
fi

echo "正在打包"
# zip -r proxy.zip proxy64.exe proxy32.exe shiningbt.mobileconfig
zip -r proxy.zip proxy32.exe proxy

echo "清理无用数据"
rm -r proxy32.exe proxy
