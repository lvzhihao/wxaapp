package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/kataras/iris"
	"github.com/spf13/viper"
	"github.com/uber-go/zap"
)

type Session struct {
	SessionId string
	WxSession Jscode2Session
	Data      map[string]string
}

type Jscode2Session struct {
	Expires_in  int64
	Openid      string
	Session_key string
}

type SessionStorage interface {
	Get(string) (Session, error)
	GetByOpenId(string) (Session, error)
	Set(string, Session) error
	Destroy(string) error
}

type TestStorage struct {
	Data       map[string]Session
	DataOpenId map[string]Session
	Lk         sync.Mutex
	SessionStorage
}

func (this *TestStorage) Get(sessionId string) (Session, error) {
	this.Lk.Lock()
	defer this.Lk.Unlock()
	var sess Session
	var ok bool
	if sess, ok = this.Data[sessionId]; ok {
		return sess, nil
	} else {
		return sess, fmt.Errorf("Don't Exists")
	}
}

func (this *TestStorage) GetByOpenId(openId string) (Session, error) {
	this.Lk.Lock()
	defer this.Lk.Unlock()
	var sess Session
	var ok bool
	if sess, ok = this.DataOpenId[openId]; ok {
		return sess, nil
	} else {
		return sess, fmt.Errorf("Don't Exists")
	}
}

func (this *TestStorage) Set(sessionId string, sess Session) error {
	this.Lk.Lock()
	defer this.Lk.Unlock()
	this.Data[sessionId] = sess
	this.DataOpenId[sess.WxSession.Openid] = sess
	return nil
}

func (this *TestStorage) Destroy(sessionId string) error {
	this.Lk.Lock()
	defer this.Lk.Unlock()
	var sess Session
	var ok bool
	if sess, ok = this.Data[sessionId]; ok {
		delete(this.Data, sessionId)
		delete(this.DataOpenId, sess.WxSession.Openid)
	}
	return nil
}

var ss = NewTestStorage()

func NewTestStorage() *TestStorage {
	ts := &TestStorage{}
	ts.Data = make(map[string]Session, 0)
	ts.DataOpenId = make(map[string]Session, 0)
	return ts
}

func getWxSession(code string) (Jscode2Session, int64, error) {
	var ret Jscode2Session
	if code == "" {
		return ret, 500001, fmt.Errorf("Code为空")
	}
	urls := url.Values{}
	urls.Add("appid", viper.GetString("app_id"))
	urls.Add("secret", viper.GetString("app_secret"))
	urls.Add("js_code", code)
	urls.Add("grant_type", "authorization_code")
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?" + urls.Encode())
	res, err := http.Get(url)
	if err != nil {
		return ret, 500002, err
	}
	info, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return ret, 500003, err
	}
	logger.Debug("jscode2session raw body:", zap.String("body", string(info)))
	err = json.Unmarshal(info, &ret)
	if err != nil || ret.Openid == "" {
		return ret, 500004, fmt.Errorf(string(info))
	}
	logger.Debug("jscode2session result:", zap.String("code", code), zap.Object("return", ret))
	return ret, 0, nil

}

func getSession(code string) (Session, int64, error) {
	var sess Session
	res, errCode, err := getWxSession(code)
	log.Println(res, errCode, err)
	if err != nil {
		return sess, errCode, err
	}
	sess, err = ss.GetByOpenId(res.Openid)
	if err != nil {
		//如果不存在
		sess.SessionId = RandStr(168)
		sess.WxSession = res
		sess.Data = make(map[string]string, 0)
		sess.Data["createTime"] = time.Now().String() //todo
		logger.Debug("CreateSession", zap.Object("session", sess))
		ss.Set(sess.SessionId, sess)
	} else if sess.WxSession.Session_key != res.Session_key {
		// wxSession expire,  update wxSession and clear data
		sess.WxSession = res
		ss.Set(sess.SessionId, sess)
	} else {
		//update expire
		//todo
	}
	return sess, 0, nil
}

//todo 常规则情况下请使用GetSessionId
func GetOpenId(ctx *iris.Context) {
	logger.Debug("form data", zap.String("code", ctx.FormValue("code")))
	sess, errCode, err := getSession(ctx.FormValue("code"))
	if err != nil {
		Failure(ctx, errCode, err)
	} else {
		Success(ctx, map[string]string{"openid": sess.WxSession.Openid})
	}
}

func GetSessionId(ctx *iris.Context) {
	logger.Debug("form data", zap.String("code", ctx.FormValue("code")))
	sess, errCode, err := getSession(ctx.FormValue("code"))
	if err != nil {
		Failure(ctx, errCode, err)
	} else {
		Success(ctx, map[string]string{"sessionId": sess.SessionId})
	}
}

func GetSession(ctx *iris.Context) {
	logger.Debug("form data", zap.String("session_id", ctx.FormValue("session_id")))
	var sess Session
	var err error
	var sessionId = ctx.FormValue("session_id")
	if sessionId == "" {
		Failure(ctx, 500001, fmt.Errorf("session_id为空"))
	}
	sess, err = ss.Get(sessionId)
	if err != nil {
		Failure(ctx, 400001, err)
	} else {
		Success(ctx, sess.Data)
	}
}
