#!/bin/bash -e

# echo "正在生成64位的Proxy"
# GOOS=windows GOARCH=amd64 go build  -o proxy64.exe proxy.go

echo "正在生成32位的Proxy"
GOOS=windows GOARCH=386 go build  -o proxy32.exe proxy.go

echo "正在打包"
# zip -r proxy.zip proxy64.exe proxy32.exe shiningbt.mobileconfig
zip -r proxy.zip proxy32.exe shiningbt.mobileconfig

echo "清理无用数据"
rm -r proxy32.exe