package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris"
	"github.com/spf13/viper"
	"github.com/uber-go/zap"
)

//session struct
type Session struct {
	SessionId string
	UserInfo  map[string]interface{}
	WxSession Jscode2Session
	Data      map[string]interface{}
}

//wx jssession success code
type Jscode2Session struct {
	Expires_in  int64
	OpenId      string
	Session_key string
}

//todo
type WxUserInfoWater struct {
	Appid     string `json:"appid"`
	Timestamp int64  `json:"timestamp"`
}

//session storage interface
type SessionStorage interface {
	Get(string) (Session, error)
	GetByOpenId(string) (Session, error)
	Set(string, Session) error
	Destroy(string) error
}

//session test //demo, faeture support : redis memcached mysql
type TestStorage struct {
	Data       map[string]Session
	DataOpenId map[string]Session
	Lk         sync.Mutex
	SessionStorage
}

//get session by sessionId
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

//get session by openid
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

//set session by sessionid
func (this *TestStorage) Set(sessionId string, sess Session) error {
	this.Lk.Lock()
	defer this.Lk.Unlock()
	this.Data[sessionId] = sess
	this.DataOpenId[sess.WxSession.OpenId] = sess
	return nil
}

//destroy session by sessionId
func (this *TestStorage) Destroy(sessionId string) error {
	this.Lk.Lock()
	defer this.Lk.Unlock()
	var sess Session
	var ok bool
	if sess, ok = this.Data[sessionId]; ok {
		delete(this.Data, sessionId)
		delete(this.DataOpenId, sess.WxSession.OpenId)
	}
	return nil
}

//global sesion storage
var ss = NewTestStorage()

//demo storage
func NewTestStorage() *TestStorage {
	ts := &TestStorage{}
	ts.Data = make(map[string]Session, 0)
	ts.DataOpenId = make(map[string]Session, 0)
	return ts
}

//fetch wx session by oauth code
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
	if err != nil || ret.OpenId == "" {
		return ret, 500004, fmt.Errorf(string(info))
	}
	logger.Debug("jscode2session result:", zap.String("code", code), zap.Object("return", ret))
	return ret, 0, nil
}

//获取session
func getSession(code string) (Session, int64, error) {
	var sess Session
	res, errCode, err := getWxSession(code)
	log.Println(res, errCode, err)
	if err != nil {
		return sess, errCode, err
	}
	sess, err = ss.GetByOpenId(res.OpenId)
	if err != nil {
		//如果不存在
		sess.SessionId = RandStr(168)
		sess.WxSession = res
		sess.UserInfo = make(map[string]interface{}, 0)
		sess.Data = make(map[string]interface{}, 0)
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
		Success(ctx, map[string]string{"openid": sess.WxSession.OpenId})
	}
}

//获取sessionid api
func GetSessionId(ctx *iris.Context) {
	logger.Debug("form data", zap.String("code", ctx.FormValue("code")))
	sess, errCode, err := getSession(ctx.FormValue("code"))
	if err != nil {
		Failure(ctx, errCode, err)
	} else {
		Success(ctx, map[string]string{"sessionId": sess.SessionId})
	}
}

//获取session api
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

func PutSession(ctx *iris.Context) {
	logger.Debug("form data", zap.String("session_id", ctx.PostValue("session_id")), zap.String("data", ctx.FormValue("data")))
	var sessionId = ctx.FormValue("session_id")
	sess, err := ss.Get(sessionId)
	if err != nil {
		Failure(ctx, 400001, err)
	} else {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(ctx.FormValue("data")), &data); err != nil {
			Failure(ctx, 400002, err)
		} else {
			sess.Data = data
			if err := ss.Set(sessionId, sess); err != nil {
				Failure(ctx, 400003, err)
			} else {
				Success(ctx, data)
			}
		}
	}
}

func GetUserInfo(ctx *iris.Context) {
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
		Success(ctx, sess.UserInfo)
	}
}

func PutUserInfo(ctx *iris.Context) {
	logger.Debug("form data", zap.Object("formObject", ctx.FormValues()))
	var sessionId = ctx.FormValue("session_id")
	sess, err := ss.Get(sessionId)
	if err != nil {
		Failure(ctx, 400001, err)
		return
	}
	var rawData = ctx.FormValue("rawData")
	var signature = ctx.FormValue("signature")
	var encryptedData = ctx.FormValue("encryptedData")
	var iv = ctx.FormValue("iv")
	checkSign := fmt.Sprintf("%x", sha1.Sum([]byte(rawData+sess.WxSession.Session_key)))
	if checkSign != signature {
		logger.Debug("CheckSign error", zap.String("checkSign", checkSign))
		Failure(ctx, 400004, fmt.Errorf("checkSign error"))
		return
	}
	ciphertext, _ := base64.StdEncoding.DecodeString(encryptedData)
	key, _ := base64.StdEncoding.DecodeString(sess.WxSession.Session_key)
	ivHex, _ := base64.StdEncoding.DecodeString(iv)
	block, err := aes.NewCipher(key)
	if err != nil {
		Failure(ctx, 500003, err)
		return
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		Failure(ctx, 500005, fmt.Errorf("ciphertext is not a multiple of the block size"))
		return
	}
	mode := cipher.NewCBCDecrypter(block, ivHex)
	mode.CryptBlocks(ciphertext, ciphertext)
	//微信返回值会有控制字符，golang json默认不过滤这些，需要处理掉
	jsstr := strings.Map(func(r rune) rune {
		if r >= 32 && r < 127 {
			return r
		}
		return -1
	}, string(ciphertext))
	var info map[string]interface{}
	err = json.Unmarshal([]byte(jsstr), &info)
	if err != nil {
		Failure(ctx, 500005, err)
		return
	}
	sess.UserInfo = info
	err = ss.Set(sessionId, sess)
	if err != nil {
		Failure(ctx, 500005, err)
	} else {
		Success(ctx, sess.UserInfo)
	}
}
