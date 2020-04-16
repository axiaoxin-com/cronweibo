// 普通cron

package cronweibo

import (
	"fmt"

	"github.com/axiaoxin-com/logging"
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
			logging.Errorw(nil, "Cron AddFunc return error", "err", err, "appname", c.appname)
		} else {
			logging.Debugw(nil, "Cron AddFunc successful", "jobName", job.Name, "jobSchedule", job.Schedule, "appname", c.appname, "entryID", entryID)
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
