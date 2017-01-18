package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/kataras/iris"
	"github.com/spf13/viper"
	"github.com/uber-go/zap"
)

type Jscode2Session struct {
	Expires_in  int64
	Openid      string
	Session_key string
}

func GetOpenId(ctx *iris.Context) {
	logger.Debug("form data", zap.String("code", ctx.FormValue("code")))
	var code = ctx.FormValue("code")
	if code == "" {
		Failure(ctx, 500001, fmt.Errorf("Code为空"))
		return
	}
	urls := url.Values{}
	urls.Add("appid", viper.GetString("app_id"))
	urls.Add("secret", viper.GetString("app_secret"))
	urls.Add("js_code", code)
	urls.Add("grant_type", "authorization_code")
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?" + urls.Encode())
	res, err := http.Get(url)
	if err != nil {
		Failure(ctx, 500002, err)
		return
	}
	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Failure(ctx, 500003, err)
		return
	}
	var ret Jscode2Session
	err = json.Unmarshal(info, &ret)
	if err != nil {
		Failure(ctx, 500004, err)
		return
	}
	logger.Debug("jscode2session result:", zap.String("code", code), zap.Object("return", ret))
	Success(ctx, map[string]string{"Openid": ret.Openid})
	return
}
