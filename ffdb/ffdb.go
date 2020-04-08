package ffdb

import (
	. "github.com/coolwhite520/alipayserver/ffdata"
	"encoding/json"
	"fmt"
	log "github.com/jeanphorn/log4go"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"sync"
)

var instance *FFdb
var once sync.Once

type FFdb struct {
	db *leveldb.DB
}

func GetInstance() *FFdb {
	once.Do(func() {
		db, err := leveldb.OpenFile("./db", nil)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		instance = &FFdb{}
		instance.db = db
	})
	return instance
}

func (f *FFdb) CloseDb() error {
	return f.db.Close()
}

/**
根据抓取的页面姓名写入到数据库中
*/
func (f *FFdb) InsertUserLoginName(paymentid int, userName string) error {
	key := fmt.Sprintf("login.%d", paymentid)
	err := f.db.Put([]byte(key), []byte(userName), nil)
	if err != nil {
		return err
	}
	return nil
}

/**
根据paymentid获取用户状态
*/
func (f *FFdb) SearchUserLoginName(paymentid int) (string, error) {
	key := fmt.Sprintf("login.%d", paymentid)
	value, err := f.db.Get([]byte(key), nil)
	if err != nil {
		return "", err
	}
	return string(value), nil
}

/**
删除用户登录信息
*/
func (f *FFdb) DelUserLoginName(paymentid int) error {
	key := fmt.Sprintf("login.%d", paymentid)
	err := f.db.Delete([]byte(key), nil)
	if err != nil {
		return err
	}
	return nil
}

/**
根据paymentid获取二维码
*/
func (f *FFdb) SearchQrCode(paymentid int) (string, error) {

	key := fmt.Sprintf("qrcode.%d", paymentid)
	value, err := f.db.Get([]byte(key), nil)
	if err != nil {
		return "", err
	}
	return string(value), nil
}

/**
根据paymentid 插入二维码
*/
func (f *FFdb) InsertQrCode(paymentid int, qrcode string) error {
	key := fmt.Sprintf("qrcode.%d", paymentid)
	err := f.db.Put([]byte(key), []byte(qrcode), nil)
	if err != nil {
		return err
	}
	return nil
}

/**
根据paymentid 删除二维码
*/
func (f *FFdb) DelQrCode(paymentid int) error {
	key := fmt.Sprintf("qrcode.%d", paymentid)
	err := f.db.Delete([]byte(key), nil)
	if err != nil {
		return err
	}
	return nil
}

/**
插入一条收款信息的记录
*/
func (f *FFdb) InsertNewPayRecord(payData PayRecordData) error {
	payData.Sended = false
	key := fmt.Sprintf("payrecord.%d.%s", payData.Paymentid, payData.Paycode)
	value, err := json.Marshal(payData)
	if err != nil {
		return err
	}
	err = f.db.Put([]byte(key), value, nil)
	if err != nil {
		return err
	}
	return nil
}

//批量插入
func (f *FFdb) InsertNewPayRecords(payRecords []PayRecordData) error {
	for _, item := range payRecords {
		err := f.InsertNewPayRecord(item)
		if err != nil {
			return err
		}
	}
	return nil
}

/**
跟新记录状态为已发送到服务端
*/
func (f *FFdb) UpdatePayRecordSended(payData PayRecordData) error {
	payData.Sended = true
	key := fmt.Sprintf("payrecord.%d.%s", payData.Paymentid, payData.Paycode)
	value, err := json.Marshal(payData)
	if err != nil {
		return err
	}
	err = f.db.Put([]byte(key), value, nil)
	if err != nil {
		return err
	}
	return nil
}

/**
查询这个id没有发送的记录
*/
func (f *FFdb) SelectPayRecordNoSend(paymentid int) ([]PayRecordData, error) {
	var vecPayRecord []PayRecordData
	selectStartKey := fmt.Sprintf("payrecord.%d.", paymentid)
	//lastChar :=  []byte(selectStartKey)[len(selectStartKey)-1] + 1
	//selectLimitKey := fmt.Sprintf("%s%c", selectStartKey[:len(selectStartKey) -1] , int(lastChar))
	iter := f.db.NewIterator(util.BytesPrefix([]byte(selectStartKey)), nil)
	for iter.Next() {
		value := iter.Value()
		var data PayRecordData
		err := json.Unmarshal(value, &data)
		if err != nil {
			log.LOGGER("APP").Error(err)
			continue
		}
		if !data.Sended {
			vecPayRecord = append(vecPayRecord, data)
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, err
	}
	return vecPayRecord, nil
}

/**
查询这个id 所有的交易
*/
func (f *FFdb) SelectPayRecordsBy(paymentid int) ([]PayRecordData, error) {
	vecPayRecord := make([]PayRecordData, 0)
	selectStartKey := fmt.Sprintf("payrecord.%d.", paymentid)
	//lastChar :=  []byte(selectStartKey)[len(selectStartKey)-1] + 1
	//selectLimitKey := fmt.Sprintf("%s%c", selectStartKey[:len(selectStartKey) -1] , int(lastChar))
	iter := f.db.NewIterator(util.BytesPrefix([]byte(selectStartKey)), nil)
	for iter.Next() {
		value := iter.Value()
		var data PayRecordData
		err := json.Unmarshal(value, &data)
		if err != nil {
			log.LOGGER("APP").Error(err)
			continue
		}
		vecPayRecord = append(vecPayRecord, data)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, err
	}
	return vecPayRecord, nil
}

func (f *FFdb) SelectAllPayRecordNoSend() ([]PayRecordData, error) {
	vecPayRecord := make([]PayRecordData, 0)
	selectStartKey := fmt.Sprintf("payrecord.")
	//lastChar :=  []byte(selectStartKey)[len(selectStartKey)-1] + 1
	//selectLimitKey := fmt.Sprintf("%s%c", selectStartKey[:len(selectStartKey) -1] , int(lastChar))
	iter := f.db.NewIterator(util.BytesPrefix([]byte(selectStartKey)), nil)
	for iter.Next() {
		value := iter.Value()
		var data PayRecordData
		err := json.Unmarshal(value, &data)
		if err != nil {
			continue
		}
		vecPayRecord = append(vecPayRecord, data)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil, err
	}
	return vecPayRecord, nil
}
