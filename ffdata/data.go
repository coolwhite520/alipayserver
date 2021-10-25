package ffdata

import (
	"alipayserver/ffrsa"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/tebeka/selenium"
	"io"
	"os"
)

// FFChannelTag k : paymentid, v:paymentid 这么创建便于发送消息，知道哪个webpage退出
type FFChannelTag struct {
	ChQrcode   chan string //二维码的通道
	ChQuit     chan int    //主线程收到退出消息的通道
	ChSelfExit chan int    //自己退出的消息通道
	WebPage    selenium.WebDriver
}

// PayRecordData 存储在数据库中的值
type PayRecordData struct {
	Paymentid int    `json:"paymentid"`
	Money     int    `json:"money"`
	Source    int    `json:"source"`
	Time      int64  `json:"time"`
	Paycode   string `json:"paycode"`
	Sended    bool   `json:"sended"`
}

//http传输中的值
type PayRecordProtocol struct {
	Paymentid int    `json:"paymentid"`
	Money     int    `json:"money"`
	Source    int    `json:"source"`
	Time      int64  `json:"time"`
	Paycode   string `json:"paycode"`
}

//登录成功后存储在db中的结构
type PayUserLoginData struct {
	Paymentid int    `json:"paymentid"`
	UserName  string `json:"username"`
}

//http RSA 加解密 结构
type PayHttpRsaProtocal struct {
	Param string `json:"param"`
}

//Qrcode结构体 req
type PayQrcodeReq struct {
	Paymentid int    `json:"paymentid"`
	Qrcode    string `json:"qrcode"`
	Message   string `json:"message,omitempty"`
}

//标准response结构体
type PayStdRes struct {
	Paymentid int    `json:"paymentid,omitempty"`
	Success   int    `json:"success"`
	Message   string `json:"message"`
	Qrcode    string `json:"qrcode,omitempty"`
}

//lua解析后返回的结构体类型
type ParseHtmlRecord struct {
	Paymentid int
	Date      string
	Time      string
	TradeNo   string
	Money     string
	Status    string
}

type ParseHtmlRecords struct {
	Records []ParseHtmlRecord
}

type PayUpdateVersion struct {
	Version string `json:"version"`
	FileUrl string `json:"fileurl"`
	Md5     string `json:"md5"`
}

//序列化
func MakeJson(obj interface{}) (string, error) {
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(obj)
	if err != nil {
		return "", err
	}
	return bf.String(), nil
}

func MakeRsaBytes(jsonStr string) ([]byte, error) {
	//序列化后进行rsa加密
	encodeStr, err := ffrsa.GetInstance().EncodeWithRemotePubKey(jsonStr)
	if err != nil {
		return nil, err
	}
	param := PayHttpRsaProtocal{Param: encodeStr}
	bytesData, err := json.Marshal(param)
	if err != nil {
		return nil, err
	}
	return bytesData, nil
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func GetFileMd5(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("os Open error")
		return "", err
	}
	md5 := md5.New()
	_, err = io.Copy(md5, file)
	if err != nil {
		fmt.Println("io copy error")
		return "", err
	}
	md5Str := hex.EncodeToString(md5.Sum(nil))
	return md5Str, nil
}
