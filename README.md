# cronweibo

## 简介

提供简单的 API 便于快速开发定时发送微博的应用，比如定时抓取图片后发送到微博、定时获取特定数据并将其保存到微博等。

使用 cronweibo 创建一个定时微博应用只需 4 个步骤：

0. 传入配置实例化cronweibo
1. 编写生成微博内容的函数实例化微博任务
2. 注册任务
3. 运行服务

微博任务(`WeiboJob`)，包含任务名称(`Name`)，执行周期(`Schedule`)和生成微博内容的函数(`Run`)等信息。

将微博任务注册到 cronweibo 服务后，cronweibo 启动后会将所有注册的任务按其执行周期定时执行该任务中的任务函数，并将其返回的内容发送到微博。

任务函数的定义为`func() (string, io.Reader)`，返回微博文本内容和图片。

可以通过配置开启HTTP接口来调用任务便于调试。

## 特性

提供友好的 API 帮助你快速实现定时发送微博的服务

支持通过 HTTP 方法调用定时任务便于调试，HTTP 支持 BasicAuth

支持注册微博登录验证码破解方法全自动获取微博 Access Token

## 安装

```
go get -u github.com/axiaoxin-com/cronweibo
```

## 在线文档

<https://godoc.org/github.com/axiaoxin-com/cronweibo>

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

// 定时发送hello world文字到微博示例
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
		AppName:           "example",
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
		Schedule: "@every 2m", // 每2分钟一次
		Run:      f,
	}

	// 将任务注册到cronweibo
	c.RegisterWeiboJobs(helloWorldJob)

	// 启动
	c.Start()
}
```

## 他们在用

- [v-bot](https://github.com/axiaoxin-com/v-bot)
