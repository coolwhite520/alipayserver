package fflua

import (
	"errors"
	"github.com/coolwhite520/alipayserver/ffluabase"
	"github.com/coolwhite520/alipayserver/tools"
	"github.com/yuin/gluamapper"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FFLua struct {
	L *lua.LState
}

var instance *FFLua
var once sync.Once

func GetInstance() *FFLua {
	once.Do(func() {
		instance = &FFLua{}
	})
	return instance
}

//加载文件
func (self *FFLua) LoadFile(filename string) error {
	self.L = lua.NewState()
	self.L.PreloadModule("ffluabase", ffluabase.Loader)
	if err := self.L.DoFile(filename); err != nil {
		return err
	}
	return nil
}

//把参数传递给lua，让lua脚本进行解析
func (self *FFLua) InvokeParseRecords(paymentid int, html string) ([]PayRecordData, bool, error) {

	if err := self.L.CallByParam(lua.P{
		Fn:      self.L.GetGlobal("ParseRecords"),
		NRet:    2,
		Protect: true,
	}, lua.LNumber(paymentid), lua.LString(html)); err != nil {
		return nil, false, err
	}
	ret := self.L.Get(-1) // returned second value
	rs := self.L.Get(-2)  // returned first value

	if rs.Type() == lua.LTNil {
		return nil, lua.LVAsBool(ret), errors.New("html页面存在变化，请及时更新lua文件。")
	}
	var records ParseHtmlRecords
	if err := gluamapper.Map(rs.(*lua.LTable), &records); err != nil {
		return nil, lua.LVAsBool(ret), err
	}

	var vecRecord []PayRecordData
	for _, v := range records.Records {
		dateStr := strings.ReplaceAll(v.Date, ".", "-")
		tradeNOStr := tools.GetUsefulEnStr(v.TradeNo)
		if strings.ContainsAny(v.Money, "+") && v.Status == "交易成功" {
			moneyStr := strings.ReplaceAll(v.Money, "+", "")
			moneyStr = strings.ReplaceAll(moneyStr, " ", "")
			moneyF, _ := strconv.ParseFloat(moneyStr, 32)
			moneyF = math.Trunc(moneyF*1e2+0.5) * 1e-2 //四舍五入，先乘以100 然后除以100
			money := int(moneyF * 100)
			ts, _ := time.ParseInLocation("2006-01-02 15:04:05", dateStr+" "+v.Time+":00", time.Local)
			data := PayRecordData{
				Paymentid: paymentid,
				Money:     money,
				Source:    1,
				Time:      ts.Unix(),
				Paycode:   tradeNOStr,
				Sended:    false,
			}
			vecRecord = append(vecRecord, data)
		}
	}
	self.L.Pop(2)

	return vecRecord, lua.LVAsBool(ret), nil
}

//把参数传递给lua，让lua脚本进行解析
func (self *FFLua) InvokeParseLoginUser(paymentid int, html string) (*PayUserLoginData, bool, error) {

	var loginData PayUserLoginData
	if err := self.L.CallByParam(lua.P{
		Fn:      self.L.GetGlobal("ParseLoginUser"),
		NRet:    2,
		Protect: true,
	}, lua.LNumber(paymentid), lua.LString(html)); err != nil {
		return nil, false, err
	}
	ret := self.L.Get(-1) // returned second value
	rs := self.L.Get(-2)  // returned first value

	if rs.Type() == lua.LTNil {
		return nil, lua.LVAsBool(ret), errors.New("html页面存在变化，请及时更新lua文件。")
	}

	if err := gluamapper.Map(rs.(*lua.LTable), &loginData); err != nil {
		return nil, lua.LVAsBool(ret), err
	}

	self.L.Pop(2)

	return &loginData, lua.LVAsBool(ret), nil
}

//重新加载文件
func (self *FFLua) ReloadFile(filename string) error {
	self.Unload()
	self.L = lua.NewState()
	self.L.PreloadModule("ffluabase", ffluabase.Loader)
	if err := self.L.DoFile(filename); err != nil {
		return err
	}
	return nil
}

//卸载模块
func (self *FFLua) Unload() {
	if self.L != nil {
		self.L.Close()
		self.L = nil
	}
}
