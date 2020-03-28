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
		WeiboAppkey:      appkey,
		WeiboAppsecret:   appsecret,
		WeiboUsername:    username,
		WeiboPasswd:      passwd,
		WeiboRedirecturi: redirecturi,
		WeiboSecurityURL: securityURL,
		Location:         loc,
		HTTPServerAddr:   ":2222",
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
