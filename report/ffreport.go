package report

import (
	"bytes"
	. "github.com/coolwhite520/alipayserver/ffdata"
	"github.com/coolwhite520/alipayserver/ffdb"
	log "github.com/sirupsen/logrus"
	//log "github.com/jeanphorn/log4go"
	"github.com/Unknwon/goconfig"
	"io/ioutil"
	"net/http"
	"sync"
	"unsafe"
)

var cfg *goconfig.ConfigFile

type Report struct {
}

var once sync.Once
var instance *Report

func GetInstance() *Report {
	once.Do(func() {
		instance = &Report{}
		cfg, _ = goconfig.LoadConfigFile("./config.ini")
	})
	cfg.Reload()
	return instance
}

func post(bytesData []byte, url string) error {
	//创建可读的
	reader := bytes.NewReader(bytesData)
	request, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	//byte数组直接转成string，优化内存
	str := (*string)(unsafe.Pointer(&respBytes))
	//log.LOGGER("APP").Info(*str)
	log.WithFields(log.Fields{
		"ServerResponse": true,
		"ReqUrl":         url,
	}).Info(*str)

	return nil
}

func (r *Report) PostLogIn(userInfo PayUserLoginData) (bool, error) {
	data := PayUserLoginData{Paymentid: userInfo.Paymentid, UserName: userInfo.UserName}
	jsonStr, _ := MakeJson(data)

	log.WithFields(log.Fields{
		"funcName": "PostLogIn",
		"json":     jsonStr,
	}).Info()

	bytesData, err := MakeRsaBytes(jsonStr)
	if err != nil {
		return false, err
	}
	//创建可读的
	loginUrl, err := cfg.GetValue("RemoteServer", "LoginUrl")
	if err != nil {
		log.WithFields(log.Fields{
			"funcName": "PostLogIn.cfg.GetValue",
		}).Error(err)
		return false, err
	}
	err = post(bytesData, loginUrl)
	if err != nil {
		return false, err
	}
	return true, nil
}

//二次验证二维码发送
func (r *Report) PostSecondQrcode(qrinfo PayQrcodeReq) (bool, error) {

	jsonStr, _ := MakeJson(qrinfo)
	log.WithFields(log.Fields{"funcName": "PostSecondQrcode", "json": jsonStr}).Info()

	bytesData, err := MakeRsaBytes(jsonStr)
	if err != nil {
		return false, err
	}
	//创建可读的
	secondQrUrl, err := cfg.GetValue("RemoteServer", "SecondQrcodeUrl")
	if err != nil {
		log.WithFields(log.Fields{"funcName": "PostSecondQrcode.cfg.GetValue"}).Error(err.Error())
		return false, err
	}
	err = post(bytesData, secondQrUrl)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Report) PostRecords(records []PayRecordData) (bool, error) {
	bSuccess := true
	var errAll error
	for _, record := range records {
		data := PayRecordProtocol{record.Paymentid, record.Money, record.Source, record.Time, record.Paycode}
		jsonStr, _ := MakeJson(data)
		log.WithFields(log.Fields{"funcName": "PostRecords", "json": jsonStr}).Info()
		bytesData, err := MakeRsaBytes(jsonStr)
		if err != nil {
			return false, err
		}
		tradeUrl, err := cfg.GetValue("RemoteServer", "TradeUrl")
		if err != nil {
			log.WithFields(log.Fields{"funcName": "PostRecords.cfg.GetValue"}).Error(err.Error())
			return false, err
		}
		//创建可读的
		err = post(bytesData, tradeUrl)
		if err != nil {
			bSuccess = false
			errAll = err
			log.WithFields(log.Fields{"funcName": "PostRecords.post"}).Error(err.Error())
		} else {
			_ = ffdb.GetInstance().UpdatePayRecordSended(record)
		}
	}
	return bSuccess, errAll
}

func (r *Report) PostLogOut(paymentid int, title string) (bool, error) {
	data := struct {
		Paymentid int
		Title     string
	}{
		Paymentid: paymentid,
		Title:     title,
	}
	jsonStr, _ := MakeJson(data)
	log.WithFields(log.Fields{"funcName": "PostLogOut", "json": jsonStr}).Info()
	bytesData, err := MakeRsaBytes(jsonStr)
	if err != nil {
		return false, err
	}
	logoutUrl, err := cfg.GetValue("RemoteServer", "LogoutUrl")
	if err != nil {
		log.WithFields(log.Fields{
			"funcName": "PostLogOut.cfg.GetValue",
		}).Error(err)
		return false, err
	}
	//创建可读的
	err = post(bytesData, logoutUrl)
	if err != nil {
		return false, err
	}
	return true, nil
}
