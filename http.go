// http handler funcs

package cronweibo

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// HandlerAuth 为 http.HandlerFunc 包一层 basic auth
func HandlerAuth(handler http.HandlerFunc, username, passwd string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rUsername, rPasswd, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(rUsername), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(rPasswd), []byte(passwd)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="cronweibo"`)
			w.WriteHeader(401)
			w.Write([]byte("You are Unauthorized to access the application.\n"))
			return
		}
		handler(w, r)
	}
}

// weiboJobHandlerFactory 将WeiboJob生产为httpserver的handler
func (c *CronWeibo) weiboJobHandlerFactory(weiboJob WeiboJob) http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		log.Println("[INFO] cronweibo handler weibojob", weiboJob.Name, "run by", r.RemoteAddr, c.appname)
		// 指定任务获取微博内容
		text, pic := weiboJob.Run()
		// 判断文本中是否存在安全域名，没有则添加到文本内容中
		if !strings.Contains(text, c.securityURL) {
			text = text + "\n" + c.securityURL
		}
		// 检查是否更新token
		if err := c.UpdateToken(); err != nil {
			log.Println("[ERROR] weibocron UpdateToken error for job", weiboJob.Name, err)
			fmt.Fprintln(w, err)
			return
		}
		// 发送微博
		resp, err := c.weibo.StatusesShare(c.token.AccessToken, text, pic)
		if err != nil {
			log.Println("[ERROR] weibocron StatusesShare error for job", weiboJob.Name, err, resp)
			fmt.Fprintln(w, err)
			return
		}
		weiboURL := "http://weibo.com/" + resp.User.ProfileURL
		response := fmt.Sprintf(`<p>%sweibo任务: %s 执行完成. 访问 <a href="%s">%s</a> 查看详情</p>`, c.appname, weiboJob.Name, weiboURL, weiboURL)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, response)
		log.Println("[INFO] cronweibo handler weibojob", weiboJob.Name, "done.", c.appname)
		return
	}
	return handler
}

// cronJobHandlerFactory 将CronJob生产为httpserver的handler
func (c *CronWeibo) cronJobHandlerFactory(cronJob CronJob) http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		log.Println("[INFO] cronweibo handler cronjob", cronJob.Name, "run by", r.RemoteAddr, c.appname)
		// 执行任务
		cronJob.Run()
		response := fmt.Sprintf(`<p>%scron任务: %s 执行完成.</p>`, c.appname, cronJob.Name)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, response)
		log.Println("[INFO] cronweibo handler cronjob", cronJob.Name, "done.", c.appname)
		return
	}
	return handler
}
