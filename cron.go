// 普通cron

package cronweibo

import (
	"fmt"
	"log"

	"github.com/robfig/cron/v3"
)

// CronJob 默认的普通定时任务
type CronJob struct {
	Schedule string       // 定时任务表达式，同WeiboJob
	Name     string       // 任务名称
	Run      cron.FuncJob // 需要执行的普通任务函数
}

// RegisterCronJobs 注册普通的cron任务
func (c *CronWeibo) RegisterCronJobs(cronJobs ...CronJob) {
	handlersList := ""
	for _, job := range cronJobs {
		// 注册定时任务
		if entryID, err := c.cron.AddFunc(job.Schedule, job.Run); err != nil {
			log.Println("[ERROR] cronweibo add cron normal func error:", err, c.appname)
		} else {
			log.Println("[DEBUG] cronweibo added cron normal func", job.Name, "as", job.Schedule, c.appname, entryID)
		}
		// 注册HTTP接口
		if c.httpServer != nil {
			handleFunc := c.cronJobHandlerFactory(job)
			if c.basicAuthUsername != "" && c.basicAuthPasswd != "" {
				handleFunc = HandlerAuth(handleFunc, c.basicAuthUsername, c.basicAuthPasswd)
			}
			c.httpServer.HandleFunc("/cron/"+job.Name, handleFunc)
			handlersList += fmt.Sprintf(`<li><a href="/cron/%s" target="blank">%s</a></li>`, job.Name, job.Name)
		}
	}
	c.cronjobHTML += handlersList
}
