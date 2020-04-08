package ffparsepage

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/Lofanmi/pinyin-golang/pinyin"
	"github.com/Unknwon/goconfig"
	"github.com/axgle/mahonia"
	"github.com/coolwhite520/alipayserver/ffdb"
	"github.com/coolwhite520/alipayserver/fflua"
	"github.com/coolwhite520/alipayserver/report"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	log "github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	port = 9515
)

var mu sync.Mutex
var service *selenium.Service
var seleniumPath, homePage string
var cfg *goconfig.ConfigFile

func init() {
	var err error
	cfg, err = goconfig.LoadConfigFile("./config.ini")
	if err != nil {
		log.WithFields(log.Fields{"funcName": "LoadConfigFile"}).Fatal(err.Error())
		return
	}
}

func StartServer() {
	var err error
	seleniumPath, err = cfg.GetValue("Chrome", "DriverPath")
	if err != nil {
		log.WithFields(log.Fields{"GetValue": "DriverPath"}).Fatal(err.Error())
		return
	}
	homePage, err = cfg.GetValue("Chrome", "HomePage")
	if err != nil {
		log.WithFields(log.Fields{"GetValue": "HomePage"}).Fatal(err.Error())
		return
	}
	opts := []selenium.ServiceOption{}
	//selenium.SetDebug(true)
	service, err = selenium.NewChromeDriverService(seleniumPath, port, opts...)
	if nil != err {
		log.WithFields(log.Fields{"funcName": "NewChromeDriverService", "port": port}).Fatal(err.Error())
		return
	}

	log.WithFields(log.Fields{}).Info("@@@@ChromeDriverService start success.")

}

func StopServer() {
	err := service.Stop()
	if err != nil {
		log.WithFields(log.Fields{"funcName": "StopServer"}).Fatal(err.Error())
	} else {
		log.WithFields(log.Fields{}).Info("@@@@ChromeDriverService stop success.")
	}
}

func GetFirstQrcode(paymentid int, webPage selenium.WebDriver) (string, error) {

	//var arrCss []struct {
	//	Css   string
	//	Value string
	//}
	//
	//arrCss = append(arrCss, struct {
	//	Css   string
	//	Value string
	//}{Css: selenium.ByCSSSelector, Value: "canvas.barcode"})
	//
	//arrCss = append(arrCss, struct {
	//	Css   string
	//	Value string
	//}{Css: selenium.ByID, Value: "J-qrcode-img"})
	//
	//arrCss = append(arrCss, struct {
	//	Css   string
	//	Value string
	//}{Css: selenium.ByID, Value: "J-qrcode-body"})
	//
	//arrCss = append(arrCss, struct {
	//	Css   string
	//	Value string
	//}{Css: selenium.ByID, Value: "J-qrcode"})
	//
	//for _, item := range arrCss {
	//	qrcodeElment, err := webPage.FindElement(item.Css, item.Value)
	//	if err != nil {
	//		continue
	//	}
	//	time.Sleep(time.Millisecond * 100)
	//	imgBytes, err := qrcodeElment.Screenshot(false)
	//	aliQrcode, err := decodeQrString(imgBytes)
	//	if err != nil {
	//		continue
	//	}
	//	return aliQrcode, nil
	//}
	for i := 0; i < 3; i++ {
		imgBytes, err := webPage.Screenshot()
		if err != nil {
			return "", err
		}
		aliPayQrCode, err := decodeQrString(imgBytes)
		if err != nil {
			webPage.Refresh()
			time.Sleep(2000 * time.Millisecond)
		} else {
			return aliPayQrCode, nil
		}
	}
	qrcodeElment, err := webPage.FindElement(selenium.ByCSSSelector, "#J-qrcode-body")
	if err != nil {
		return "", err
	}
	imgBytes, err := qrcodeElment.Screenshot(false)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(imgBytes), nil
}

func GetSecondQrcode(paymentid int, webPage selenium.WebDriver) (string, error) {

	jsStr := "document.documentElement.scrollLeft = 0;"
	webPage.ExecuteScript(jsStr, nil)
	for i := 0; i < 3; i++ {
		imgBytes, err := webPage.Screenshot()
		if err != nil {
			return "", err
		}
		aliPayQrCode, err := decodeQrString(imgBytes)
		if err != nil {
			webPage.Refresh()
			time.Sleep(1000 * time.Millisecond)
			jsStr := "document.documentElement.scrollLeft = 0;"
			webPage.ExecuteScript(jsStr, nil)
		} else {
			return aliPayQrCode, nil
		}
	}

	qrcodeElment, err := webPage.FindElement(selenium.ByCSSSelector, "body")
	if err != nil {
		return "", err
	}
	imgBytes, err := qrcodeElment.Screenshot(false)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(imgBytes), nil
}

//http请求
// ALIPAYJSESSIONID : RZ11NXzgr6XghFKnjgXhjfr6Ch63SXauthRZ11GZ00

func GetAlipayHtmlContextByCookies(method string, urlVal string, cookieItems []selenium.Cookie, cookies []*http.Cookie) ([]byte, []*http.Cookie, error) {

	client := &http.Client{}
	var req *http.Request
	urlArr := strings.Split(urlVal, "?")
	if len(urlArr) == 2 {
		urlVal = urlArr[0] + "?" + getParseParam(urlArr[1])
	}
	req, _ = http.NewRequest(method, urlVal, nil)

	for _, cookieItem := range cookieItems {
		bFind := false
		for _, cookie := range cookies {
			if cookieItem.Name == cookie.Name {
				bFind = true
			}
		}
		if bFind {
			continue
		}
		//cookie := &http.Cookie{
		//	Name:    cookieItem.Name,
		//	Value:   cookieItem.Value,
		//	Path:    cookieItem.Path,
		//	Domain:  cookieItem.Domain,
		//	Expires: time.Unix(int64(cookieItem.Expiry), 0),
		//	Secure:  cookieItem.Secure,
		//}
		//req.AddCookie(cookie)
		cookieStr := fmt.Sprintf("%s=%s", cookieItem.Name, cookieItem.Value)
		fmt.Println(cookieStr)
		req.Header.Add("Cookie", cookieStr)
	}

	for _, cookie := range cookies {
		cookieStr := fmt.Sprintf("%s=%s", cookie.Name, cookie.Value)
		fmt.Println(cookieStr)
		req.Header.Add("Cookie", cookieStr)
	}

	req.Header.Add("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/76.0.3809.87 Safari/537.36")

	fmt.Println(req.Cookies())
	for i := 0; i < 10; i++ {
		resp, err := client.Do(req)
		if err != nil {
			if strings.Contains(err.Error(), "Client.Timeout exceeded") || strings.Contains(err.Error(), "wsarecv: A connection attempt failed") {
				fmt.Println(err)
			} else {
				return nil, nil, err
			}
		}
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		cookies = resp.Cookies()
		return b, cookies, err
	}
	return nil, nil, errors.New("Client.Timeout exceeded")
}

//将get请求的参数进行转义
func getParseParam(param string) string {
	return url.PathEscape(param)
}

func Run(paymentid int, chObj *FFChannelTag) {
	var title string
	defer func() {
		_, _ = report.GetInstance().PostLogOut(paymentid, title)
		chObj.ChSelfExit <- paymentid
	}()
	webPagecookies, initRecords, err := OpenNewPage(paymentid, chObj)
	if err != nil {
		log.WithFields(log.Fields{"funcName": "OpenNewPage", "paymentid": paymentid}).Error(err.Error())
		return
	}
	var cookies []*http.Cookie
	var bytesHtml []byte
	for {
		bytesHtml, cookies, err = GetAlipayHtmlContextByCookies("GET", homePage, webPagecookies, cookies)
		if err != nil {
			log.WithFields(log.Fields{"funcName": "GetAlipayHtmlContextByCookies", "paymentid": paymentid, "success": false}).Error(err.Error())
			return
		}
		bodystr := mahonia.NewDecoder("gbk").ConvertString(string(bytesHtml))

		newRecords, _, err := fflua.GetInstance().InvokeParseRecords(paymentid, bodystr)
		if err != nil {
			log.WithFields(log.Fields{"funcName": "InvokeParseRecords", "paymentid": paymentid, "success": false}).Error(err.Error())
			ioutil.WriteFile(string(paymentid)+".html", bytesHtml, 777)
			return
		}

		log.WithFields(log.Fields{"FuncName": "loopParseHtml", "paymentid": paymentid, "success": true}).Info()

		//第一次认为是初始数据不写入数据库, 但是需要上报登录成功
		diffRecords := DiffNewRecords(initRecords, newRecords)
		if len(diffRecords) > 0 {
			err := ffdb.GetInstance().InsertNewPayRecords(diffRecords)
			if err != nil {
				log.WithFields(log.Fields{"funcName": "InsertNewPayRecords", "paymentid": paymentid}).Error(err.Error())
				return
			}
			initRecords = append(initRecords, diffRecords...)
		}
		needReportRecords, err := ffdb.GetInstance().SelectPayRecordNoSend(paymentid)
		if err != nil {
			log.WithFields(log.Fields{"funcName": "SelectPayRecordNoSend", "paymentid": paymentid}).Error(err.Error())
			return
		}
		if len(needReportRecords) > 0 {
			go func() {
				b, err := report.GetInstance().PostRecords(needReportRecords)
				if err == nil && b {
					return
				} else {
					if err != nil {
						log.WithFields(log.Fields{"funcName": "PostRecords", "paymentid": paymentid}).Error(err.Error())
					}
				}
			}()
		}
		span := randInt64(20, 60)
		for i := 0; i < int(span); i++ {
			select {
			case <-chObj.ChQuit:
				log.WithFields(log.Fields{"FuncName": "Run", "paymentid": paymentid, "success": true}).Info("Main thread sends notification to exit.")
				return
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}

}

/**
	创建一个新的页面
    paymentid: id
	ch ： 可写channel，负责传递qrcode
	chquit： 可读channel，负责读取外部通知退出消息
	chselftQuit : 可写channel，通知外部，自己退出
*/
func OpenNewPage(paymentid int, chObj *FFChannelTag) ([]selenium.Cookie, []PayRecordData, error) {
	//链接本地的浏览器 chrome
	caps := selenium.Capabilities{
		"browserName": "chrome",
	}
	//禁止图片加载，加快渲染速度
	imagCaps := map[string]interface{}{
		"profile.managed_default_content_settings.images": 2,
	}
	imagCaps = nil

	headless, err := cfg.GetValue("WebPage", "HeadLess")
	if err != nil {
		log.WithFields(log.Fields{"GetValue": "HeadLess"}).Fatal(err.Error())
		return nil, nil, err
	}

	var args []string
	args = append(args, "--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.100 Safari/537.36")
	args = append(args, fmt.Sprintf("--paymentid=%d", paymentid))
	if headless == "true" {
		args = append(args, "--headless")
	}

	chromeCaps := chrome.Capabilities{
		Prefs:           imagCaps,
		Path:            "",
		Args:            args,
		ExcludeSwitches: []string{"enable-automation", "enable-logging"},
	}
	//以上是设置浏览器参数
	caps.AddChrome(chromeCaps)
	// 调起chrome浏览器

	webPage, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
	if err != nil {
		log.WithFields(log.Fields{"funcName": "NewRemote", "paymentid": paymentid}).Error(err.Error())
		return nil, nil, err
	}
	//关闭一个webDriver会对应关闭一个chrome窗口
	//但是不会导致seleniumServer关闭
	defer func() {
		_ = webPage.Quit()
	}()

	_ = webPage.MaximizeWindow("")

	err = webPage.Get(homePage)
	if err != nil {
		log.WithFields(log.Fields{"funcName": "Get", "paymentid": paymentid}).Error(err.Error())
		go OpenNewPage(paymentid, chObj)
		return nil, nil, err
	}

	//打开主页面后需要获取首次登录的二维码
	aliQrcode, err := GetFirstQrcode(paymentid, webPage)
	if err != nil {
		log.WithFields(log.Fields{"funcName": "GetFirstQrcode", "paymentid": paymentid}).Error(err.Error())
		return nil, nil, err
	}
	//写入管道中需要回馈给请求方的数据
	chObj.ChQrcode <- aliQrcode

	//等待用户扫码登录
	var recvExitMsg bool
	err = webPage.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (b bool, e error) {
		if title, _ := wd.Title(); title == "我的账单 - 支付宝" || title == "安全校验 - 支付宝" {
			return true, nil
		} else {
			select {
			case <-chObj.ChQuit:
				recvExitMsg = true
				return true, nil
			default:
				return false, nil
			}
		}
	}, 4*time.Minute, 1*time.Second)

	if err != nil {

		return nil, nil, err
	}
	//主线程发送的主动退出消息
	if recvExitMsg {
		log.WithFields(log.Fields{"paymentid": paymentid, "success": true}).Info("Main thread sends notification to exit.")
		return nil, nil, err
	}
	//切换到高级用户
	SwithToPrimaryUser(paymentid, webPage)

	//准备第一次登录成功的上报
	html, err := webPage.PageSource()
	if err != nil {
		log.WithFields(log.Fields{"paymentid": paymentid}).Info(err.Error())
		return nil, nil, err
	}
	userInfo, success, err := fflua.GetInstance().InvokeParseLoginUser(paymentid, html)
	if err != nil {
		log.WithFields(log.Fields{"funcName": "InvokeParseLoginUser", "paymentid": paymentid}).Error(err.Error())
		return nil, nil, err
	}
	if !success {
		log.WithFields(log.Fields{"funcName": "InvokeParseLoginUser", "paymentid": paymentid, "success": success}).Error(err.Error())
		return nil, nil, err
	}
	//中文转一下
	dict := pinyin.NewDict()
	userInfo.UserName = dict.Abbr(userInfo.UserName, "-")
	ok, err := report.GetInstance().PostLogIn(*userInfo)
	if err != nil || !ok {
		log.WithFields(log.Fields{"funcName": "PostLogIn", "paymentid": paymentid}).Error(err.Error())
		return nil, nil, err
	}

	//初始化循环需要的变量
	bSecondCheck := false
	parseCount := 0
	var initRecords []PayRecordData

	for {
		select {
		//收到退出的信号
		case <-chObj.ChQuit:
			log.WithFields(log.Fields{"paymentid": paymentid, "success": true}).Info("Main thread sends notification to exit.")
			return nil, nil, err
		default:
			//模拟点击
			//AutoClickOperation(paymentid, webPage)
			title, _ := webPage.Title()
			log.WithFields(log.Fields{"title": title, "paymentid": paymentid}).Info("")
			//扫码已经完成，但是需要二次验证

			if title == "安全校验 - 支付宝" {

				chObj.WebPage = webPage
				aliQrcode, err := GetSecondQrcode(paymentid, webPage)
				//data:image/jpg;base64,
				if err != nil {
					log.WithFields(log.Fields{"funcName": "GetSecondQrcode", "paymentid": paymentid}).Error(err.Error())
					return nil, nil, err
				}
				//发送二次验证码
				go func() {
					for {
						b, err := report.GetInstance().PostSecondQrcode(PayQrcodeReq{
							Paymentid: paymentid,
							Qrcode:    aliQrcode,
						})
						if err == nil && b {
							log.WithFields(log.Fields{"funcName": "PostSecondQrcode", "paymentid": paymentid, "success": true}).Info("")
							break
						} else {
							if err != nil {
								log.WithFields(log.Fields{"funcName": "PostSecondQrcode", "paymentid": paymentid}).Error(err.Error())
							}
						}
						time.Sleep(2 * time.Second)
					}
				}()
				//二次验证等待跳转
				var recvExitMsg bool
				err = webPage.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (b bool, e error) {
					if title, _ := wd.Title(); title == "我的账单 - 支付宝" {
						return true, nil
					} else {
						select {
						case <-chObj.ChQuit:
							recvExitMsg = true
							return true, nil
						default:
							return false, nil
						}
					}
				}, 4*time.Minute, 1*time.Second)

				if err != nil {
					return nil, nil, err
				}
				//主线程发送的主动退出消息
				if recvExitMsg {
					log.WithFields(log.Fields{"paymentid": paymentid, "success": true}).Info("Main thread sends notification to exit.")
					return nil, nil, err
				}
				//
				html, err := webPage.PageSource()
				if err != nil {
					log.WithFields(log.Fields{"paymentid": paymentid}).Info(err.Error())
					return nil, nil, err
				}
				userInfo, success, err := fflua.GetInstance().InvokeParseLoginUser(paymentid, html)
				if err != nil {
					log.WithFields(log.Fields{"funcName": "InvokeParseLoginUser", "paymentid": paymentid}).Error(err.Error())
					return nil, nil, err
				}
				if !success {
					log.WithFields(log.Fields{"funcName": "InvokeParseLoginUser", "paymentid": paymentid, "success": success}).Error(err.Error())
					return nil, nil, err
				}
				//中文转一下
				dict := pinyin.NewDict()
				userInfo.UserName = dict.Abbr(userInfo.UserName, "-")
				ok, err := report.GetInstance().PostLogIn(*userInfo)
				if err != nil || !ok {
					log.WithFields(log.Fields{"funcName": "PostLogIn", "paymentid": paymentid}).Error(err.Error())
					return nil, nil, err
				}
				parseCount = 0
				bSecondCheck = true //经过了二次验证
			} else if title == "我的账单 - 支付宝" {

				parseCount++
				//数据解析开始
				html, err := webPage.PageSource()
				if err != nil {
					log.WithFields(log.Fields{"paymentid": paymentid}).Info(err.Error())
					return nil, nil, err
				}
				newRecords, _, err := fflua.GetInstance().InvokeParseRecords(paymentid, html)
				if err != nil {
					log.WithFields(log.Fields{"funcName": "InvokeParseRecords", "paymentid": paymentid, "success": false}).Error(err.Error())
					return nil, nil, err
				}
				//第一次认为是初始数据不写入数据库, 但是需要上报登录成功
				if parseCount == 1 {
					initRecords = newRecords
					if bSecondCheck {
						cookies, _ := webPage.GetCookies()
						return cookies, initRecords, err
					}
				} else {
					diffRecords := DiffNewRecords(initRecords, newRecords)
					if len(diffRecords) > 0 {
						err := ffdb.GetInstance().InsertNewPayRecords(diffRecords)
						if err != nil {
							log.WithFields(log.Fields{"funcName": "InsertNewPayRecords", "paymentid": paymentid}).Error(err.Error())
							return nil, nil, err
						}
						initRecords = append(initRecords, diffRecords...)
					}
					needReportRecords, err := ffdb.GetInstance().SelectPayRecordNoSend(paymentid)
					if err != nil {
						log.WithFields(log.Fields{"funcName": "SelectPayRecordNoSend", "paymentid": paymentid}).Error(err.Error())
						return nil, nil, err
					}
					if len(needReportRecords) > 0 {
						go func() {
							b, err := report.GetInstance().PostRecords(needReportRecords)
							if err == nil && b {
								return
							} else {
								if err != nil {
									log.WithFields(log.Fields{"funcName": "PostRecords", "paymentid": paymentid}).Error(err.Error())
								}
							}
						}()
					}
				}

			} else if title == "支付宝 知托付！" { //重复登陆的时候会这样哦，标题是---《支付宝 知托付！》并且存在用户名登陆
				userNameEle, err := webPage.FindElement(selenium.ByCSSSelector, "div.entry>div.state")
				if err != nil { //可能已经掉线了
					//可能已经掉线了
					log.WithFields(log.Fields{"paymentid": paymentid}).Error(err)
					return nil, nil, err
				}
				username, err := userNameEle.Text()
				if err != nil { //可能已经掉线了
					log.WithFields(log.Fields{"paymentid": paymentid}).Error(err)
					return nil, nil, err
				}
				if strings.Contains(username, "你好，") { // 这个时候说明用户重复登陆
					log.WithFields(log.Fields{"paymentid": paymentid}).Error("User repeat to login.")
					return nil, nil, err
				} else { //可能已经掉线了
					log.WithFields(log.Fields{"paymentid": paymentid}).Error("Not find userName")
					return nil, nil, err
				}
			} else { //其他错误页面
				log.WithFields(log.Fields{"PageError": "CurrentPage", "paymentid": paymentid, "title": title}).Error("CurrentPage is error, please...")
				//保留现场，不return
				return nil, nil, err
			}
			if parseCount > 0 {
				span := randInt64(50, 70)
				for i := 0; i < int(span); i++ {
					select {
					case <-chObj.ChQuit:
						log.WithFields(log.Fields{"paymentid": paymentid, "success": true}).Info("Main thread sends notification to exit.")
						return nil, nil, err
					default:
						time.Sleep(1 * time.Second)
					}
				}

			}
		}
	}
}

/**
判断某条记录是否属于数组中
*/
func IsInRecords(record PayRecordData, initRecords []PayRecordData) bool {
	for _, item := range initRecords {
		if record.Paycode == item.Paycode {
			return true
		}
	}
	return false
}

/**
查询两个数组记录的差异
*/
func DiffNewRecords(initRecords, newRecords []PayRecordData) []PayRecordData {
	var diffNewRecords []PayRecordData
	for _, item := range newRecords {
		if !IsInRecords(item, initRecords) {
			diffNewRecords = append(diffNewRecords, item)
		}
	}
	return diffNewRecords
}

/**
跳转到正常页面
*/
func ClickBtnJump2NoramlPage(webPage selenium.WebDriver) {

	btn, err := webPage.FindElement(selenium.ByLinkText, "充值记录")
	if err != nil {
		log.WithFields(log.Fields{"funcName": "ClickBtnJump2NoramlPage", "value": "充值记录"}).Error(err.Error())
		_ = webPage.Refresh()
		return
	}
	_ = btn.Click()

	//随机sleep
	span := randInt64(1*1000, 2*1000)
	time.Sleep(time.Duration(span) * time.Millisecond)

	btn, err = webPage.FindElement(selenium.ByLinkText, "交易记录")
	if err != nil {
		log.WithFields(log.Fields{"funcName": "ClickBtnJump2NoramlPage", "value": "交易记录"}).Error(err.Error())
		_ = webPage.Refresh()
		return
	}
	_ = btn.Click()
}

func SwithToPrimaryUser(paymentid int, webPage selenium.WebDriver) {
	btn2PrimaryUser, err := webPage.FindElement(selenium.ByLinkText, "切换到高级版")
	if err == nil {
		_ = btn2PrimaryUser.Click()
		log.WithFields(log.Fields{"funcName": "SwithToPrimaryUser", "value": "切换到高级版", "paymentid": paymentid}).Info("Success switch")
	}

}

func AutoClickOperation(paymentid int, webPage selenium.WebDriver) {
	//模拟滚动条操作
	jsArray := []string{"document.documentElement.scrollTop = 1000000;",
		"document.documentElement.scrollTop = 0;",
		"document.documentElement.scrollLeft = 1000000;",
		"document.documentElement.scrollLeft = 0;",
	}
	//随机sleep
	span := randInt64(1*1000, 2*1000)
	time.Sleep(time.Duration(span) * time.Millisecond)

	//2. 随机点击其他link
	randX := randInt64(0, 50)
	randY := randInt64(0, 100)

	randomNum := randInt64(4, 8)
	switch randomNum {
	case 4:
		btn, err := webPage.FindElement(selenium.ByLinkText, "充值记录")
		if err != nil {
			log.WithFields(log.Fields{"funcName": "AutoClickOperation", "value": "充值记录", "paymentid": paymentid}).Error(err.Error())
			_ = webPage.Refresh()
			return
		}
		_ = btn.MoveTo((int)(randX), (int)(randY))
		_ = btn.Click()
	case 5:
		span = randInt64(0, 3)
		jsStr := jsArray[span]
		_, _ = webPage.ExecuteScript(jsStr, nil)
	case 6:
		btn, err := webPage.FindElement(selenium.ByLinkText, "提现记录")
		if err != nil {
			log.WithFields(log.Fields{"funcName": "AutoClickOperation", "value": "提现记录", "paymentid": paymentid}).Error(err.Error())
			_ = webPage.Refresh()
			return
		}
		_ = btn.MoveTo((int)(randX), (int)(randY))
		_ = btn.Click()
	case 7:
		btn, err := webPage.FindElement(selenium.ByLinkText, "电子对账单")
		if err != nil {
			log.WithFields(log.Fields{"funcName": "AutoClickOperation", "value": "电子对账单", "paymentid": paymentid}).Error(err.Error())
			_ = webPage.Refresh()
			return
		}
		_ = btn.MoveTo((int)(randX), (int)(randY))
		_ = btn.Click()
	}

	span = randInt64(0, 3)
	jsStr := jsArray[span]
	_, _ = webPage.ExecuteScript(jsStr, nil)

	//随机sleep
	span = randInt64(2*1000, 4*1000)
	time.Sleep(time.Duration(span) * time.Millisecond)

	//点击一次交易记录
	btnTrade, err := webPage.FindElement(selenium.ByLinkText, "交易记录")
	if err != nil {
		log.WithFields(log.Fields{"funcName": "AutoClickOperation", "value": "交易记录", "paymentid": paymentid}).Error(err.Error())
		_ = webPage.Refresh()
		return
	}
	_ = btnTrade.MoveTo((int)(randX), (int)(randY))
	_ = btnTrade.Click()

	//随机sleep
	span = randInt64(2*1000, 4*1000)
	time.Sleep(time.Duration(span) * time.Millisecond)

	//模拟滚动条操作
	span = randInt64(0, 3)
	jsStr = jsArray[span]
	_, _ = webPage.ExecuteScript(jsStr, nil)

}

//生成随机数
func randInt64(min, max int64) int64 {
	if min >= max || max == 0 {
		return max
	}
	return rand.Int63n(max-min) + min
}

/**
解密图片的二维码信息
*/
func decodeQrString(imgData []byte) (string, error) {
	//s := base64.StdEncoding.EncodeToString(imgData)
	//fmt.Println(s)
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		log.WithFields(log.Fields{"funcName": "image.Decode"}).Error(err.Error())
		return "", err
	}

	// prepare BinaryBitmap
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		log.WithFields(log.Fields{"funcName": "gozxing.NewBinaryBitmapFromImage"}).Error(err.Error())
		return "", err
	}
	var hints = make(map[gozxing.DecodeHintType]interface{})
	hints[gozxing.DecodeHintType_CHARACTER_SET] = "utf-8"
	hints[gozxing.DecodeHintType_POSSIBLE_FORMATS] = gozxing.BarcodeFormat_QR_CODE
	hints[gozxing.DecodeHintType_TRY_HARDER] = true
	// decode image
	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, hints)
	if err != nil {
		log.WithFields(log.Fields{"funcName": "qrReader.Decode"}).Error(err.Error())
		return "", err
	}
	return result.GetText(), nil
}
