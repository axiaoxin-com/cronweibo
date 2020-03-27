// Package cronweibo 提供简单的方法便于快速开发定时发送微博的应用
//
// 实现一个定时发送微博的程序你只需实现具体的任务函数，
// 生成对应的任务后，将其注册到服务中，服务运行后将按注册的定时规则定时执行任务中的执行函数，并将执行结果发送到微博
package cronweibo

import (
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/axiaoxin-com/weibo"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
)

// CronWeibo 定时微博服务定义
type CronWeibo struct {
	weibo            *weibo.Weibo
	token            *weibo.TokenResp
	tokenCreatedAt   int64
	tokenUpdateMutex sync.Mutex
	securityURL      string
	cron             *cron.Cron
	weiboJobs        []WeiboJob
	location         *time.Location
}

// Config CronWeibo配置定义，New函数的参数
type Config struct {
	// 微博相关配置
	WeiboUsername       string               // 要发微博的微博登录账号（必填参数，用于模拟登录自动获取授权码）
	WeiboPasswd         string               // 要发微博的微博登录密码（必填参数，用于模拟登录
	WeiboPinCrackFuncs  []weibo.CrackPinFunc // 登录验证码破解函数（非必填）
	WeiboAppkey         string               // 微博应用的appkey（必填参数）
	WeiboAppsecret      string               // 微博应用的appsecret（必填参数）
	WeiboRedirecturi    string               // 微博应用的回调地址（必填参数）
	WeiboSecurityDomain string               // 微博应用的安全域名（必填参数）
	// cron相关配置
	Location *time.Location // 指定定时服务的时区（非必填）
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
	log.Println("[INFO] cronweibo is initializing...")
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
		weibo:       weibo,
		token:       token,
		securityURL: "http://" + config.WeiboSecurityDomain,
		cron:        c,
		location:    loc,
	}
	cw.tokenCreatedAt = cw.Now().Unix()
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
		log.Println("[INFO] cronweibo token will expire, set a new token:", token)
	}
	return nil
}

// cronFuncFactory 将WeiboJob生产为cron的FuncJob
func (c *CronWeibo) cronFuncFactory(weiboJob WeiboJob) cron.FuncJob {
	cronFunc := func() {
		log.Println("[INFO] cronweibo is doing job:", weiboJob.Name)
		// 指定任务获取微博内容
		text, pic := weiboJob.Run()
		// 判断文本中是否存在安全域名，没有则添加到文本内容中
		if !strings.Contains(text, c.securityURL) {
			text = text + " " + c.securityURL
		}
		// 检查是否更新token
		if err := c.UpdateToken(); err != nil {
			log.Println("[ERROR] weibocron UpdateToken error for job", weiboJob.Name, err)
			return
		}
		// 发送微博
		resp, err := c.weibo.StatusesShare(c.token.AccessToken, text, pic)
		if err != nil {
			log.Println("[ERROR] weibocron StatusesShare error for job", weiboJob.Name, err, resp)
		}
	}
	return cronFunc
}

// RegisterWeiboJobs 注册微博任务
func (c *CronWeibo) RegisterWeiboJobs(weiboJobs ...WeiboJob) {
	for _, job := range weiboJobs {
		// job转换为cronFunc
		cronFunc := c.cronFuncFactory(job)
		// 注册定时任务
		if err := c.cron.AddFunc(job.Schedule, cronFunc); err != nil {
			log.Println("[ERROR] cronweibo add cron func error:", err)
		} else {
			log.Println("[DEBUG] cronweibo added cron func:", job.Name, "as", job.Schedule)
		}
	}
}

// Start 启动定时微博服务
func (c *CronWeibo) Start() {
	log.Println("[INFO] cronweibo is running...")
	c.cron.Start()
	defer c.cron.Stop()
	select {}
}
