package ffautoupdate

import (
	. "alipayserver/ffdata"
	"alipayserver/fflua"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Unknwon/goconfig"
	log "github.com/sirupsen/logrus"
)

var (
	DefaultSleepTimeInterval = 10 //默认升级loop时间为10分钟
)

type ffAutoUpdate struct {
}

var once sync.Once
var instance *ffAutoUpdate

func GetInstance() *ffAutoUpdate {
	once.Do(func() {
		instance = &ffAutoUpdate{}
	})
	return instance
}

func (f *ffAutoUpdate) DoWork() {
	for {
		cfg, err := goconfig.LoadConfigFile("./config.ini")
		if err != nil {
			log.WithFields(log.Fields{"funcName": "LoadConfigFile"}).Error(err.Error())
			time.Sleep(time.Duration(DefaultSleepTimeInterval) * time.Minute)
			continue
		}
		strUrl, err := cfg.GetValue("AutoUpdate", "UpdateUrl")
		if err != nil {
			log.WithFields(log.Fields{}).Error(err)
			time.Sleep(time.Duration(DefaultSleepTimeInterval) * time.Minute)
			continue
		}

		timeInterval, err := cfg.Int64("AutoUpdate", "TimeInterval")
		if err != nil {
			log.WithFields(log.Fields{}).Error(err)
			time.Sleep(time.Duration(DefaultSleepTimeInterval) * time.Minute)
			continue
		}

		time.Sleep(time.Duration(timeInterval) * time.Minute)
		//先看看是否存在temp目录
		tempPath := "./temp"
		_, err = os.Stat(tempPath)
		if b, _ := PathExists(tempPath); !b {
			err := os.Mkdir(tempPath, 0777)
			if err != nil {
				log.WithFields(log.Fields{}).Error(err)
				continue
			}
		}

		//先下载version.json文件, 到临时目录
		tempVersionFile := "./temp/version.json"
		localVersionFile := "./version.json"

		if b, _ := PathExists(tempVersionFile); b {
			err = os.Remove(tempVersionFile)
			if err != nil {
				log.WithFields(log.Fields{}).Error(err)
				continue
			}
		}
		err = f.downloadFile(strUrl, tempVersionFile, nil)
		if err != nil {
			log.WithFields(log.Fields{}).Error(err)
			continue
		}

		newVersionObj, err := f.parseVersionJsonFile(tempVersionFile)
		if err != nil {
			log.WithFields(log.Fields{}).Error(err)
			continue
		}

		oldVersionObj := &PayUpdateVersion{}
		if ok, _ := PathExists(localVersionFile); ok {
			oldVersionObj, err = f.parseVersionJsonFile(localVersionFile)
			if err != nil {
				log.WithFields(log.Fields{}).Error(err)
				continue
			}
		} else {
			oldVersionObj.Version = ""
		}

		if newVersionObj.Version != oldVersionObj.Version {
			//下载json里面对应的文件
			localLuaFile := "./parse.lua"
			downLuaFile := "./temp/parse.lua"
			if b, _ := PathExists(downLuaFile); b {
				err := os.Remove(downLuaFile)
				if err != nil {
					log.WithFields(log.Fields{}).Error(err)
					continue
				}
			}
			err = f.downloadFile(newVersionObj.FileUrl, downLuaFile, nil)
			if err != nil {
				log.WithFields(log.Fields{}).Error(err)
				continue
			}
			//校验文件md5
			fileMd5str, err := GetFileMd5(downLuaFile)
			if err != nil {
				log.WithFields(log.Fields{}).Error(err)
				continue
			}
			if !strings.EqualFold(fileMd5str, newVersionObj.Md5) {
				log.WithFields(log.Fields{"newMd5": newVersionObj.Md5, "calMd5": fileMd5str}).Error("下载文件的md5错误。")
				continue
			}
			//校验成功了，那么需要copy到根路径并重新加载lua文件
			//先删除本地脚本文件
			if b, _ := PathExists(localLuaFile); b {
				err = os.Remove(localLuaFile)
				if err != nil {
					log.WithFields(log.Fields{}).Error(err)
					continue
				}
			}
			// 把下载的文件copy到目标路径
			_, err = CopyFile(localLuaFile, downLuaFile)
			if err != nil {
				log.WithFields(log.Fields{}).Error(err)
				continue
			} else {
				err = fflua.GetInstance().ReloadFile(localLuaFile)
				if err != nil {
					log.WithFields(log.Fields{}).Error(err)
					continue
				} else {
					log.WithFields(log.Fields{"CurrentVersion": newVersionObj.Version}).Info("AutoUpdate success")
				}
			}
			//最后把version.json也copy到路径中
			if b, _ := PathExists(localVersionFile); b {
				err := os.Remove(localVersionFile)
				if err != nil {
					log.WithFields(log.Fields{}).Error(err)
					continue
				}
			}
			_, err = CopyFile(localVersionFile, tempVersionFile)
			if err != nil {
				log.WithFields(log.Fields{}).Error(err)
				continue
			}
		}
	}
}

//获取本地verison.json的版本
func (f *ffAutoUpdate) parseVersionJsonFile(pathFileName string) (*PayUpdateVersion, error) {
	dataBytes, err := ioutil.ReadFile(pathFileName)
	if err != nil {
		return nil, err
	}
	var versionObj PayUpdateVersion
	err = json.Unmarshal(dataBytes, &versionObj)
	if err != nil {
		return nil, err
	}
	return &versionObj, nil
}

func (f *ffAutoUpdate) downloadFile(url string, localPath string, fb func(length, downLen int64)) error {
	var (
		fsize   int64
		buf     = make([]byte, 32*1024)
		written int64
	)
	tmpFilePath := localPath + ".download"
	client := new(http.Client)
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	//读取服务器返回的文件大小
	fsize, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 32)
	if err != nil {
		return err
	}
	if b, _ := PathExists(tmpFilePath); b {
		err := os.Remove(tmpFilePath)
		if err != nil {
			return err
		}
	}
	//创建文件
	file, err := os.Create(tmpFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	if resp.Body == nil {
		return errors.New("body is null")
	}
	defer resp.Body.Close()
	//下面是 io.copyBuffer() 的简化版本
	for {
		//读取bytes
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			//写入bytes
			nw, ew := file.Write(buf[0:nr])
			//数据长度大于0
			if nw > 0 {
				written += int64(nw)
			}
			//写入出错
			if ew != nil {
				err = ew
				break
			}
			//读取是数据长度不等于写入的数据长度
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
		fb(fsize, written)
	}
	if err == nil {
		_ = file.Close()
		if b, _ := PathExists(localPath); b {
			err := os.Remove(localPath)
			if err != nil {
				return err
			}
		}
		err = os.Rename(tmpFilePath, localPath)
		if err != nil {
			return err
		}
		return nil
	} else {
		return err
	}
}
