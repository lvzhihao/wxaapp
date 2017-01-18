package api

import (
	"encoding/hex"
	"math"
	"math/rand"
	"time"

	"github.com/kataras/iris"
	"github.com/uber-go/zap"
)

//zap logger
var logger = zap.New(
	zap.NewTextEncoder(zap.TextTimeFormat(time.RFC822)),
	zap.DebugLevel,
)

//为nil时则表示为系统内部错误，小程序前端获取接口errMsg为null则提示自定义错误，同示可显示logId，方便调试
var errMsgMap = map[int64]interface{}{
	500001: nil, //code为空
	500002: nil,
	500003: nil,
	500004: nil,
}

func RandStr(len int64) string {
	b := make([]byte, int(math.Ceil(float64(len)/2.0)))
	rand.Seed(time.Now().UnixNano())
	rand.Read(b)
	return hex.EncodeToString(b)[0:len]
}

func Failure(ctx *iris.Context, errCode int64, err error) {
	var logId = RandStr(16)
	errMsg, _ := errMsgMap[errCode]
	logger.Error("wxaapp api error", zap.String("logId", logId), zap.Int64("errCode", errCode), zap.String("error", err.Error()))
	ctx.JSON(200, map[string]interface{}{"data": nil, "errMsg": errMsg, "logId": logId})
}

func Success(ctx *iris.Context, data interface{}) {
	var logId = RandStr(16)
	logger.Info("wxaapp api success", zap.String("logId", logId), zap.Object("data", data))
	ctx.JSON(200, map[string]interface{}{"data": data, "errMsg": "request:ok", "logId": logId})
}
