// Package cronweibo 提供简单的 API 便于快速开发定时发送微博的应用
//
// 比如定时抓取图片后发送到微博、定时获取特定数据并将其保存到微博等。
//
// 使用 cronweibo 创建一个定时微博应用只需 4 个步骤：
//
//   0. 传入配置实例化cronweibo
//   1. 编写生成微博内容的函数实例化微博任务
//   2. 注册任务
//   3. 运行服务
//
// 微博任务(WeiboJob)，包含任务名称(Name)，执行周期(Schedule)和生成具体微博内容的函数(Run)等信息。
//
// 将微博任务注册到 cronweibo 服务后，cronweibo 启动后会将所有注册的任务按其执行周期定时执行该任务中的任务函数，并将其返回的内容发送到微博。
//
// 可以通过配置开启HTTP接口来调用任务便于调试。
package cronweibo

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/axiaoxin-com/weibo"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
)

// CronWeibo 定时微博服务定义
type CronWeibo struct {
	appname           string
	weibo             *weibo.Weibo
	token             *weibo.TokenResp
	tokenCreatedAt    int64
	tokenUpdateMutex  sync.Mutex
	securityURL       string
	cron              *cron.Cron
	weiboJobs         []WeiboJob
	location          *time.Location
	httpServer        *http.ServeMux
	httpServerAddr    string
	basicAuthUsername string
	basicAuthPasswd   string
	cronjobHTML       string
	weibojobHTML      string
}

// Config CronWeibo配置定义，New函数的参数
type Config struct {
	AppName string // 定时微博app名称（非必填）
	// 微博相关配置
	WeiboUsername      string               // 要发微博的微博登录账号（必填参数，用于模拟登录自动获取授权码）
	WeiboPasswd        string               // 要发微博的微博登录密码（必填参数，用于模拟登录
	WeiboPinCrackFuncs []weibo.CrackPinFunc // 登录验证码破解函数（非必填）
	WeiboAppkey        string               // 微博应用的 appkey （必填参数）
	WeiboAppsecret     string               // 微博应用的 appsecret （必填参数）
	WeiboRedirecturi   string               // 微博应用的回调地址（必填参数）
	WeiboSecurityURL   string               // 微博应用的安全链接（必填参数，http:// + 微博应用中配置的安全域名）
	// cron server 相关配置
	Location *time.Location // 指定定时服务的时区（非必填）

	// HTTP server 相关配置
	HTTPServerAddr    string // HTTP 服务运行地址 （非必填），设置后会运行HTTP服务提供 GET 方式请求 http://host:port/jobname 可立即执行任务
	BasicAuthUsername string // 和 BasicAuthPasswd 同时配置时，会对所有的HTTP接口进行基础认证（非必填）
	BasicAuthPasswd   string // 和 BasicAuthUsername 同时配置时，会对所有的HTTP接口进行基础认证（非必填）
}

// WeiboJobFunc 微博任务函数类型声明
// 不接收参数，返回微博文本内容和微博图片内容
type WeiboJobFunc func() (string, io.Reader)

// WeiboPinCrackFunc 微博验证码破解函数类型声明
type WeiboPinCrackFunc weibo.CrackPinFunc

// WeiboJob 微博任务定义，任务名 + 定时表达式 + 任务函数组成☝️任务
type WeiboJob struct {
	/* Schedule 格式参考：
	   Entry                  | Description                                | Equivalent To
	   @yearly (or @annually) | Run once a year, midnight, Jan. 1st        | 0 0 0 1 1 *
	   @monthly               | Run once a month, midnight, first of month | 0 0 0 1 * *
	   @weekly                | Run once a week, midnight between Sat/Sun  | 0 0 0 * * 0
	   @daily (or @midnight)  | Run once a day, midnight                   | 0 0 0 * * *
	   @hourly                | Run once an hour, beginning of hour        | 0 0 * * * *
	*/
	Schedule string       // 定时任务表达式
	Name     string       // 任务名称
	Run      WeiboJobFunc // 需要执行的微博任务函数
}

// New 创建CronWeibo实例
func New(config *Config, weiboJobs ...WeiboJob) (*CronWeibo, error) {
	log.Println("[INFO] cronweibo is initializing", config.AppName)
	// 创建weibo实例
	weibo := weibo.New(config.WeiboAppkey, config.WeiboAppsecret, config.WeiboUsername, config.WeiboPasswd, config.WeiboRedirecturi)
	// 注册验证码破解函数
	weibo.RegisterCrackPinFunc(config.WeiboPinCrackFuncs...)
	// 登录微博获取accesstoken
	if err := weibo.PCLogin(); err != nil {
		return nil, errors.Wrap(err, "cronweibo login weibo error")
	}
	code, err := weibo.Authorize()
	if err != nil {
		return nil, errors.Wrap(err, "cronweibo get authorize code error")
	}
	log.Println("[DEBUG] cronweibo get authorize code:", code)
	token, err := weibo.AccessToken(code)
	if err != nil {
		return nil, errors.Wrap(err, "cronweibo get access token error")
	}
	log.Println("[DEBUG] cronweibo get token:", token)

	// 创建带时区的cron实例
	loc := config.Location
	if loc == nil {
		loc = time.Now().Location()
	}
	c := cron.NewWithLocation(loc)

	// 创建CronWeibo
	cw := &CronWeibo{
		appname:           config.AppName,
		weibo:             weibo,
		token:             token,
		securityURL:       config.WeiboSecurityURL,
		cron:              c,
		location:          loc,
		basicAuthUsername: config.BasicAuthUsername,
		basicAuthPasswd:   config.BasicAuthPasswd,
	}
	cw.tokenCreatedAt = cw.Now().Unix()

	// 如果配置了HTTPServerAddr，会实例化http server，服务启动后会运行一个http服务提供web api执行任务
	if config.HTTPServerAddr != "" {
		cw.httpServer = http.NewServeMux()
		cw.httpServerAddr = config.HTTPServerAddr
	}
	log.Println("[INFO] cronweibo initialize successful.")
	return cw, nil
}

// Now 获取CronWeibo中的当前时间
// 应用中需要获取当前时间请使用该方法保证时间时区正确
func (c *CronWeibo) Now() time.Time {
	now := time.Now().In(c.location)
	return now
}

// UpdateToken 检查access_token是否过期，过期则更新
// 一般情况无需使用到，默认在注册任务后执行任务时会自动检查
func (c *CronWeibo) UpdateToken() error {
	// 互斥锁
	c.tokenUpdateMutex.Lock()
	defer c.tokenUpdateMutex.Unlock()
	// 判断到当前时间为止token已存在时间是否已大于其过期时间
	age := c.Now().Unix() - c.tokenCreatedAt
	log.Printf("[DEBUG] weiboclock check token age=%d, ExpiresIn=%d", age, c.token.ExpiresIn)
	// 过期则更新token
	if age >= c.token.ExpiresIn {
		if err := c.weibo.PCLogin(); err != nil {
			return errors.Wrap(err, "weiboclock UpdateToken PCLogin error")
		}
		code, err := c.weibo.Authorize()
		if err != nil {
			return errors.Wrap(err, "weiboclock UpdateToken Authorize error")
		}
		token, err := c.weibo.AccessToken(code)
		if err != nil {
			return errors.Wrap(err, "weiboclock UpdateToken AccessToken error")
		}
		c.token = token
		log.Println("[INFO] cronweibo", c.appname, "token will expire, set a new token:", token)
	}
	return nil
}

// cronFuncFactory 将WeiboJob生产为cron的FuncJob
func (c *CronWeibo) cronFuncFactory(weiboJob WeiboJob) cron.FuncJob {
	cronFunc := func() {
		log.Println("[INFO] cronweibo cron job", weiboJob.Name, "run.", c.appname)
		// 指定任务获取微博内容
		text, pic := weiboJob.Run()
		// 判断文本中是否存在安全域名，没有则添加到文本内容中
		if !strings.Contains(text, c.securityURL) {
			text = text + "\n" + c.securityURL
		}
		// 检查是否更新token
		if err := c.UpdateToken(); err != nil {
			log.Println("[ERROR] weibocron UpdateToken error for job", weiboJob.Name, err, c.appname)
			return
		}
		// 发送微博
		resp, err := c.weibo.StatusesShare(c.token.AccessToken, text, pic)
		if err != nil {
			log.Println("[ERROR] weibocron StatusesShare error for job", weiboJob.Name, err, resp, c.appname)
			return
		}
		log.Println("[INFO] cronweibo cron job", weiboJob.Name, "done.", c.appname)
	}
	return cronFunc
}

// RegisterWeiboJobs 注册微博任务
func (c *CronWeibo) RegisterWeiboJobs(weiboJobs ...WeiboJob) {
	handlersList := ""
	for _, job := range weiboJobs {
		// job转换为cronFunc
		cronFunc := c.cronFuncFactory(job)
		// 注册定时任务
		if err := c.cron.AddFunc(job.Schedule, cronFunc); err != nil {
			log.Println("[ERROR] cronweibo add cron weibo func error:", err, c.appname)
		} else {
			log.Println("[DEBUG] cronweibo added cron weibo func", job.Name, "as", job.Schedule, c.appname)
		}
		// 注册HTTP接口
		if c.httpServer != nil {
			handleFunc := c.weiboJobHandlerFactory(job)
			if c.basicAuthUsername != "" && c.basicAuthPasswd != "" {
				handleFunc = HandlerAuth(handleFunc, c.basicAuthUsername, c.basicAuthPasswd)
			}
			c.httpServer.HandleFunc("/weibo/"+job.Name, handleFunc)
			log.Println("[DEBUG] cronweibo added http weibo func", job.Name, c.appname)
			handlersList += fmt.Sprintf(`<li><a href="/weibo/%s" target="blank">%s</a></li>`, job.Name, job.Name)
		}
	}
	c.weibojobHTML += handlersList
}

// addr转url
func (c *CronWeibo) addr2URL(addr string) string {
	s := strings.Split(addr, ":")
	if len(s) == 2 {
		port := s[1]
		return fmt.Sprintf("http://%s:%s", weibo.RealIP(), port)
	}
	return addr
}

// Start 启动定时微博服务
func (c *CronWeibo) Start() {
	// 启动 HTTP server
	if c.httpServer != nil {
		go func() {
			// 添加首页导航页面
			index := "<p>weibo jobs</p><ul>" + c.weibojobHTML + "</ul>"
			index += "<p>cron jobs</p><ul>" + c.cronjobHTML + "</ul>"
			c.httpServer.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprintln(w, index)
				return
			})
			log.Println("[INFO] cronweibo start http server on", c.addr2URL(c.httpServerAddr), c.appname)
			if err := http.ListenAndServe(c.httpServerAddr, c.httpServer); err != nil {
				log.Println("[ERROR] cronweibo start http server failed.", err, c.appname)
			}
		}()
	}
	log.Println("[INFO] cronweibo is starting", c.appname)
	c.cron.Start()
	defer c.cron.Stop()
	select {}
}

// WeiboClient 返回当前 weibo client
func (c *CronWeibo) WeiboClient() *weibo.Weibo {
	return c.weibo
}

// Token 返回当前 token
func (c *CronWeibo) Token() *weibo.TokenResp {
	return c.token
}
