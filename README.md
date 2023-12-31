# 迷你跳板机 mjump
```                   
  /\/\   (_) _   _  _ __ ___   _ __  
 /    \  | || | | || '_ \` _ \| '_ \
/ /\/\ \ | || |_| || | | | | || |_) |
\/    \/_/ | \__,_||_| |_| |_|| .__/ 
       |__/                   |_|    
```
> 用于小型企业或者个人服务器集中管理，集中管理服务器密码，实现一个账号登录多个服务器。类似于堡垒机，可二次开发自己所需功能.

[![GoDoc](https://godoc.org/github.com/gliderlabs/ssh?status.svg)](https://godoc.org/github.com/gliderlabs/ssh) 
[![CircleCI](https://img.shields.io/circleci/project/github/gliderlabs/ssh.svg)](https://circleci.com/gh/gliderlabs/ssh)
[![Go Report Card](https://goreportcard.com/badge/github.com/gliderlabs/ssh)](https://goreportcard.com/report/github.com/gliderlabs/ssh) 
[![OpenCollective](https://opencollective.com/ssh/sponsors/badge.svg)](#sponsors)
[![Email Updates](https://img.shields.io/badge/updates-subscribe-yellow.svg)](https://app.convertkit.com/landing_pages/243312)
## Usage
```
// 修改配置文件无需重启服务.
edit config.json
{
    "users": [
	// 跳板机用户信息填写，assets 资产信息的ID，手动授权用户可访问的主机
        { "id":"1","username":"root", "password": "123456","public":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDHG96CC5Us1OYwRrSRKJTzcJpvhzUT+fznLg0dXpSej2wmkfmLh+9Tni5udi4PddzeQgEBJM0wyK93Z3s1ha/Cq9i0DLfGdKsOMP0D5RToEFvvHAVbOOW6ZSsx8MfnovZwLaMcPW1wI0UN5ScSjGp/yLxSzX3TGREQ68VC01pYqcX3Bnxfo+vL6zUVCDTmn3ochLrSp5zohQ1iIMG/A8/36v/+4krMMNCYTSVezt2Uh/cEF80o4g19sth6lKcYB0rAESLo8GytzKbWKvgSOyia3mep08iy7o206Y3YPAlsNQFqRL9rvlP6dLrdwXFca9j0qNfkY7oLb5n7iqjBW/kb","assets":[ "1","2","3","5" ]}
    ],
    "hosts": [
	// 资产信息添加,ID 和name 是唯一的，不能重复
        { "id":"1","name":"文341","username":"root","ip":"192.168.1.101","password":"123456", "prot":"22","isactive":"0" ,"ftpdir":"","privateKey":""},
        { "id":"2","name":"摄入地方2","username":"root","ip":"192.168.1.147","password":"123456", "prot":"22","isactive":"0" ,"ftpdir":"/","privateKey":""},
     ...
    ],
    "listen": {
        "host":"0.0.0.0",
        "port": "2222"
    }
}

./mjump
```
Docker Run
```
docker pull ffeenn/mjump
docker run -itd -p 2222:2222 -v ./config.json:/config.json ffeenn/mjump
```
## Make
```
yum -y install git go
git clone https://github.com/ffeenn/mjump.git
cd mjump
go env -w GOPROXY=https://proxy.golang.com.cn,direct
go env -w CGO_ENABLED=0
GOOS=linux GOARCH=amd64 go build cmd/mjump.go
```
## Examp
![image](https://github.com/ffeenn/mjump/assets/43292253/d7b36f5f-d4e7-4237-82dd-ca92920c6c99)

![image](https://github.com/ffeenn/mjump/assets/43292253/daad93e6-1503-41b3-a9e9-b0869582e4f0)
![image](https://github.com/ffeenn/mjump/assets/43292253/16659027-f590-474f-8827-fd444ddbcfdb)



