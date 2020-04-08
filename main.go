package main

import (
	"bytes"
	"encoding/json"
	"github.com/coolwhite520/alipayserver/ffautoupdate"
	"github.com/coolwhite520/alipayserver/ffdb"
	"github.com/coolwhite520/alipayserver/fflua"
	"github.com/coolwhite520/alipayserver/ffparsepage"
	"github.com/coolwhite520/alipayserver/ffrsa"
	"github.com/coolwhite520/alipayserver/ip"
	"github.com/coolwhite520/alipayserver/logcuthook"
	"github.com/coolwhite520/alipayserver/loghook"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

var (
	    app    *iris.Application
	webPageMap = make(map[int]*FFChannelTag)
)

func init() {
	logcuthook.ConfigLocalFilesystemLogger("./logs", "mylog", time.Hour*24*60, time.Hour*24)
	log.AddHook(loghook.NewContextHook())
	log.SetFormatter(&log.TextFormatter{TimestampFormat: "2006-01-02 15:04:05"})
	//加载配置文件
	err := fflua.GetInstance().LoadFile("./parse.lua")
	if err != nil {
		log.WithFields(log.Fields{}).Error(err)
		return
	}
	go WatchAllPageSelfExitEvent()
	go ffautoupdate.GetInstance().DoWork()
	go listenSignal()
}

func main() {

	ffparsepage.StartServer()
	defer ffparsepage.StopServer()
	app = iris.Default()
	app.Use(LoggerMiddleware)
	//拉起操作
	app.Post("/run", HandleRun)
	//关闭
	app.Post("/unbind", HandleUnbind)
	// 查看订单
	app.Get("/search", HandleSearch)
	// 重新获取二维码
	app.Post("/get2qrcode", HandleGetSecondQrcode)

	err := app.Run(iris.Addr(":4000"))
	if err != nil {
		log.WithFields(log.Fields{}).Error(err)
	}
}

func listenSignal() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case <-sigs:
		c := exec.Command("taskkill.exe", "/f", "/im", "chrome.exe")
		_ = c.Start()
		os.Exit(0)
	}
}

// LoggerMiddleware 日志中间件
func LoggerMiddleware(ctx iris.Context) {
	p := ctx.Request().URL.Path
	method := ctx.Request().Method
	start := time.Now()
	fields := make(map[string]interface{})
	fields["title"] = "G@@dL#ck."
	fields["ip"] = ip.RemoteIp(ctx.Request())
	fields["method"] = method
	fields["url"] = ctx.Request().URL.String()
	//fields["proto"] = ctx.Request().Proto
	//fields["header"] = ctx.Request().Header
	fields["user_agent"] = ctx.Request().UserAgent()
	//fields["x_request_id"] = ctx.GetHeader("X-Request-Id")

	// 如果是POST/PUT请求，并且内容类型为JSON，则读取内容体
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		body, err := ioutil.ReadAll(ctx.Request().Body)
		if err == nil {
			defer ctx.Request().Body.Close()
			buf := bytes.NewBuffer(body)
			ctx.Request().Body = ioutil.NopCloser(buf)
			//fields["content_length"] = ctx.GetContentLength()
			//fields["body"] = string(body)
		}
	}
	log.WithFields(fields).Infof("[http] %s-%s-%s-%d(0ms)",
		p, ctx.Request().Method, ip.RemoteIp(ctx.Request()), ctx.ResponseWriter().StatusCode())
	ctx.Next()

	//下面是返回日志
	fields["res_status"] = ctx.ResponseWriter().StatusCode()
	if ctx.Values().GetString("out_err") != "" {
		fields["out_err"] = ctx.Values().GetString("out_err")
	}
	fields["res_length"] = ctx.ResponseWriter().Header().Get("size")
	if v := ctx.Values().Get("res_body"); v != nil {
		if b, ok := v.([]byte); ok {
			fields["res_body"] = string(b)
		}
	}
	fields["uid"] = ctx.Values().GetString("uid")
	timeConsuming := time.Since(start).Nanoseconds() / 1e6
	log.WithFields(fields).Infof("[http] %s-%s-%s-%d(%dms)",
		p, ctx.Request().Method, ip.RemoteIp(ctx.Request()), ctx.ResponseWriter().StatusCode(), timeConsuming)
}

//监听哪个page的退出
func WatchAllPageSelfExitEvent() {
	for {
		time.Sleep(time.Second)
		for k, chObj := range webPageMap {
			select {
			case v := <-chObj.ChSelfExit:
				log.WithFields(log.Fields{"funcName": "WatchAllPageSelfExitEvent", "paymentid": v, "exit": true}).Info("")
				delete(webPageMap, k)
			default:
				continue
			}
		}
	}
}

// 第一次传递的二次验证二维码不好看，重新获取二维码
func HandleGetSecondQrcode(ctx context.Context) {
	var jsonObj PayHttpRsaProtocal
	err := ctx.ReadJSON(&jsonObj)
	if err != nil {
		str, _ := MakeJson(PayStdRes{Paymentid: 0, Success: 0, Message: "Parameters must be passed through JSON.", Qrcode: ""})
		bytesData, _ := MakeRsaBytes(str)
		_, _ = ctx.WriteString(string(bytesData))
		log.WithFields(log.Fields{"json": str}).Error(str)
		return
	} else {
		jsonStr, err := ffrsa.GetInstance().DecodeWithLocalPrivateKey(jsonObj.Param)
		if err != nil {
			str, _ := MakeJson(PayStdRes{Paymentid: 0, Success: 0, Message: "Rsa decode Err", Qrcode: ""})
			bytesData, _ := MakeRsaBytes(str)
			_, _ = ctx.WriteString(string(bytesData))
			log.WithFields(log.Fields{"json": str}).Error(err.Error())
			return
		}
		var obj struct{ Paymentid int }
		err = json.Unmarshal([]byte(jsonStr), &obj)
		if err != nil {
			str, _ := MakeJson(PayStdRes{Paymentid: -1, Success: 0, Message: "Unmarshal decode Err"})
			bytesData, _ := MakeRsaBytes(str)
			_, _ = ctx.WriteString(string(bytesData))
			log.WithFields(log.Fields{"json": str}).Error(str)
			return
		}
		paymentid := obj.Paymentid
		qrcode, err := ffparsepage.GetSecondQrcode(paymentid, webPageMap[paymentid].WebPage)
		if err != nil {
			str, _ := MakeJson(PayStdRes{Paymentid: -1, Success: 0, Message: "Get qrcode err."})
			bytesData, _ := MakeRsaBytes(str)
			_, _ = ctx.WriteString(string(bytesData))
			log.WithFields(log.Fields{"json": str}).Error(str)
			return
		}
		jsonStr, _ = MakeJson(PayStdRes{Paymentid: paymentid, Success: 1, Message: "success", Qrcode: qrcode})
		dataBytes, _ := MakeRsaBytes(jsonStr)
		_, _ = ctx.WriteString(string(dataBytes))
		log.WithFields(log.Fields{}).Info(jsonStr)
		return
	}
}

/**
拉起
*/
func HandleRun(ctx context.Context) {
	var jsonObj PayHttpRsaProtocal
	err := ctx.ReadJSON(&jsonObj)
	if err != nil {
		str, _ := MakeJson(PayStdRes{Paymentid: 0, Success: 0, Message: "Parameters must be passed through JSON.", Qrcode: ""})
		bytesData, _ := MakeRsaBytes(str)
		_, _ = ctx.WriteString(string(bytesData))
		log.WithFields(log.Fields{"json": str}).Error(str)
		return
	} else {
		jsonStr, err := ffrsa.GetInstance().DecodeWithLocalPrivateKey(jsonObj.Param)
		if err != nil {
			str, _ := MakeJson(PayStdRes{Paymentid: 0, Success: 0, Message: "Rsa decode Err", Qrcode: ""})
			bytesData, _ := MakeRsaBytes(str)
			_, _ = ctx.WriteString(string(bytesData))
			log.WithFields(log.Fields{"json": str}).Error(err.Error())
			return
		}
		var obj struct{ Paymentid int }
		err = json.Unmarshal([]byte(jsonStr), &obj)
		if err != nil {
			str, _ := MakeJson(PayStdRes{Paymentid: -1, Success: 0, Message: "Unmarshal decode Err"})
			bytesData, _ := MakeRsaBytes(str)
			_, _ = ctx.WriteString(string(bytesData))
			log.WithFields(log.Fields{"json": str}).Error(str)
			return
		}
		paymentid := obj.Paymentid
		if webPageMap[paymentid] != nil {
			jsonStr, _ = MakeJson(PayStdRes{Paymentid: paymentid, Success: 1, Message: "The paymentid is running."})
			dataBytes, _ := MakeRsaBytes(jsonStr)
			_, _ = ctx.WriteString(string(dataBytes))
			log.WithFields(log.Fields{}).Info(jsonStr)
			return
		}

		chTag := &FFChannelTag{
			ChQrcode:   make(chan string),
			ChQuit:     make(chan int),
			ChSelfExit: make(chan int),
		}

		webPageMap[paymentid] = chTag
		go ffparsepage.Run(paymentid, chTag)
		qrcode := <-chTag.ChQrcode
		jsonStr, _ = MakeJson(PayStdRes{Paymentid: paymentid, Success: 1, Message: "success", Qrcode: qrcode})
		dataBytes, _ := MakeRsaBytes(jsonStr)
		_, _ = ctx.WriteString(string(dataBytes))
		log.WithFields(log.Fields{}).Info(jsonStr)
		return
	}
}

/**
解绑定操作
*/
func HandleUnbind(ctx context.Context) {
	var jsonObj PayHttpRsaProtocal
	err := ctx.ReadJSON(&jsonObj)
	if err != nil {
		str, _ := MakeJson(PayStdRes{Paymentid: 0, Success: 0, Message: "Parameters must be passed through JSON.", Qrcode: ""})
		bytesData, _ := MakeRsaBytes(str)
		_, _ = ctx.WriteString(string(bytesData))
		log.WithFields(log.Fields{"json": str}).Error(err.Error())
		return
	} else {
		jsonStr, err := ffrsa.GetInstance().DecodeWithLocalPrivateKey(jsonObj.Param)
		if err != nil {
			str, _ := MakeJson(PayStdRes{Paymentid: 0, Success: 0, Message: "Rsa decode Err", Qrcode: ""})
			bytesData, _ := MakeRsaBytes(str)
			_, _ = ctx.WriteString(string(bytesData))
			log.WithFields(log.Fields{"json": str}).Error(str)
			return
		}
		var obj struct{ Paymentid int }
		err = json.Unmarshal([]byte(jsonStr), &obj)
		if err != nil {
			str, _ := MakeJson(PayStdRes{Paymentid: 0, Success: 0, Message: "Unmarshal decode Err", Qrcode: ""})
			bytesData, _ := MakeRsaBytes(str)
			_, _ = ctx.WriteString(string(bytesData))
			log.WithFields(log.Fields{"json": str}).Error(str)
			return
		}
		paymentid := obj.Paymentid
		ch := webPageMap[paymentid]
		if ch != nil {
			jsonStr, _ = MakeJson(PayStdRes{Paymentid: paymentid, Success: 1, Message: "Unbind success", Qrcode: ""})
			dataBytes, _ := MakeRsaBytes(jsonStr)
			_, _ = ctx.WriteString(string(dataBytes))
			log.WithFields(log.Fields{}).Info(jsonStr)
			go func() {
				ch.ChQuit <- paymentid
			}()
		} else {
			jsonStr, _ = MakeJson(PayStdRes{Paymentid: paymentid, Success: 1, Message: "Unbind but not find this paymentid", Qrcode: ""})
			dataBytes, _ := MakeRsaBytes(jsonStr)
			_, _ = ctx.WriteString(string(dataBytes))
			log.WithFields(log.Fields{}).Info(jsonStr)
		}
		return
	}
}

/**
通过paymentid查询所有订单
*/
func HandleSearch(ctx context.Context) {
	paymentid, err := ctx.URLParamInt("paymentid")
	if err != nil {
		_, _ = ctx.WriteString(err.Error())
		return
	}
	arr, _ := ffdb.GetInstance().SelectPayRecordsBy(paymentid)
	str, _ := MakeJson(arr)
	_, _ = ctx.WriteString(str)
	return
}
