# cronweibo

## 简介

该包提供定时发送微博的封装，提供简单的方法便于快速开发定时发送微博的应用，比如定时抓取图片后发送到微博、定时获取特定数据并将其保存到微博等。

通过定义微博任务(`WeiboJob`)，其中包含任务名称(`Name`)，执行周期(`Schedule`)和具体执行的任务函数(`Run`)等信息。

将微博任务注册到 cronweibo 服务后，cronweibo 启动后会将所有注册的任务按其执行周期定时执行该任务中的任务函数，并将其返回的内容发送到微博。

任务函数的定义为`func() (string, io.Reader)`，返回微博文本内容和图片。

## 安装

```
go get -u github.com/axiaoxin-com/cronweibo
```

## 在线文档

<https://pkg.go.dev/github.com/axiaoxin-com/cronweibo?tab=doc>

## 使用示例

一个定时发送hello world到微博的应用:

[example/hello_world.go](/example/helloworld.go)

```golang
package main

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/axiaoxin-com/cronweibo"
)

func main() {
	// 从环境变量获取配置信息
	appkey := os.Getenv("weibo_app_key")
	appsecret := os.Getenv("weibo_app_secret")
	username := os.Getenv("weibo_username")
	passwd := os.Getenv("weibo_passwd")
	redirecturi := os.Getenv("weibo_redirect_uri")
	securityURL := os.Getenv("weibo_security_url")

	// 创建配置
	loc, _ := time.LoadLocation("Asia/Shanghai")
	config := &cronweibo.Config{
		WeiboAppkey:       appkey,
		WeiboAppsecret:    appsecret,
		WeiboUsername:     username,
		WeiboPasswd:       passwd,
		WeiboRedirecturi:  redirecturi,
		WeiboSecurityURL:  securityURL,
		Location:          loc,
		HTTPServerAddr:    ":2222",
		BasicAuthUsername: "admin",
		BasicAuthPasswd:   "admin",
	}

	// 创建定时微博服务
	c, err := cronweibo.New(config)
	if err != nil {
		log.Fatal(err)
	}

	// 定义helloworld_job的任务函数
	f := func() (string, io.Reader) {
		return "hello world", nil
	}
	// 创建任务
	helloWorldJob := cronweibo.WeiboJob{
		Name:     "helloworld",
		Schedule: "0 */2 * * * *", // 每2分钟一次
		Run:      f,
	}

	// 将任务注册到cronweibo
	c.RegisterWeiboJobs(helloWorldJob)

	// 启动
	c.Start()
}
```
