package ffrsa

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	log "github.com/jeanphorn/log4go"
	"sync"
)

const (
	publicKeyStr = `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAjHq8hdAMkqyL6VFVZEE/h42Am03YFqxPooYE9cv7nx1qbW0l181qswNp/Rc8Atd9Uw593ZHe8G0XCXzrxTFtFYHBOx2KacIKO78HECKSR+V4pZeWTO6+zB4NwBJIfVjtPvd0L6ae9YdqQ1/m3Ddo4xxOWDVvAnfj47FNURAtaYYLxEoHPH17BCiIAU/fqXgsZctv8eK/OCJVRAZR/NBnX41bziNXs1qJKAtrXBh27pNanIcSQNUZogE4vYhtI4TBOx5cLq3p/dOewF4VSgvLly8BsvwQsd+hoH2J9QiYcmu4vkj16U4ZnG/sXLYRar14w3lvYAR43aGwUQFENy4sowIDAQAB`
	privateKeyStr = `MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCMeryF0AySrIvpUVVkQT+HjYCbTdgWrE+ihgT1y/ufHWptbSXXzWqzA2n9FzwC131TDn3dkd7wbRcJfOvFMW0VgcE7HYppwgo7vwcQIpJH5Xill5ZM7r7MHg3AEkh9WO0+93Qvpp71h2pDX+bcN2jjHE5YNW8Cd+PjsU1REC1phgvESgc8fXsEKIgBT9+peCxly2/x4r84IlVEBlH80GdfjVvOI1ezWokoC2tcGHbuk1qchxJA1RmiATi9iG0jhME7Hlwuren9057AXhVKC8uXLwGy/BCx36GgfYn1CJhya7i+SPXpThmcb+xcthFqvXjDeW9gBHjdobBRAUQ3LiyjAgMBAAECggEAfh2iKi/BWdx2LxzJoJvDQRqUHtkf6hr/01SmF1TtrMdnyJ14n+jWpaem+4RkZ9i1tl8IOGbA0u/dArOXpwzcdbZrl3rJzHBbZ4/z81RWJx2n1mHkmWSs/ertRUYktgOj2ielszHO+3Z6u8nZJFLKYzoCS8aMxpaDjOKcGu5/Fn/fTW1xbrDb0cLn10UC1NFxf+1N6nxQx2lpB2zU563V4ulz6TOSWTbzhDHz+Z8M2dvj/YmH7b27C5A153eDmsK02+p4rcKYJJhk6aGPWg2vxOVqohwjWXgnELmjPBfkC1FddbVvDLJ9NoNq35zQyn744LTjaJwi57tXgRVE4tiEYQKBgQDOcodJEzL7nHl7dOTi8vpzraMrGo5k873kBV1A0hI8cY+hsVyxJFPhe3Ao8U86t8o2RLVqNz9a8td5xptCFv7N/YtInDV9rtxEaQ+CecWGv68trvwlPMuLLs4HaldZgs879FWIuTjm9IiT8iL3ZgMtBo0fywn0g9GIsTXkeQYUlQKBgQCuMrVLU7rrKT4fdSCTGHVxxf+567wsKhFIW8CYo9ANcYkodiqxSzIcQITyRaq7awwxVn1X8is+lTdrWwwZg0wv2SwF37s2aSy+JVz64Cj8p6ZUHQhNYmfjGPkGogkMzC/329jqnIW50DoLbIICQ7t7aOeg7FUyEFb4zgRUaX72VwKBgQCEa5f1OnhoRezyUH+ERyB41LalrkoJ3QI/YWV4b4a8+IaNy4tOIKQhLKE70dTlMABnoNjQwZFE0CGAffEWilN0Y55Uu8Xf0Sp+Qvls0pjtnsRJxET9yaoxo5sJzJ8szigJJ+c5ZL1n/xAf/Z+KGwgMeHhLHVxMV8UuLaUI4MwSpQKBgFl+GZ+iW4QWwyCcJukLBsM1qoRUtG6oCgr3gEGq3ucZLhLIanR9IjhbeTGSUv1IKJJsUabN9iOHu1n/A39T8d8fHjvCTkbMz7y8K0GDyerLImtSIwLGjIKiQMWEl0uo1vfqTU1yrkzgygE2dLA/JeFn6p/E6pv2wwe9p4HKVPyvAoGAGkud7yQXSfXRovQ6Lffh1hDwWbEucbjK8RmpVCF7MTAi9agEi3NcBIZ8JQvMydu4Jq4CjqM+cjDLs4zmszvrbItt1sywultD1MjSAayMXd70Db8ZPFN2HVH032GDJx6fPDS2nh6Me22rTMtQnIpLWc8BIh54svdZVq8O3Zbuwsk=`
	remotePublickeyStr = `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAiu70uOgfrmgo49CETq+Zs+VTl/3IoSJlFUnoCT5+K2OH0LxarkLKdGUakvypE6KXtq+dNtp/OZUAJBJp4aFXrc9ZafCQmp+ioHyY2q11qu3cIHpRRDCCZSmCPACge4p7zKd6TmWmCKfXEF+BEOwvHc5buetD6pJxhpCZQoTRgbZ9TzHpW5nIqNfwTZOtbOcqKiqqTojQ0woMP7lY/2kE/UcBOEoZOW3ZVCQWC7H+nQXSHVu5IeZkGUe0ad7ih5fwk7wjBnMc0KIEEYlyLg672XY3Omd1O0DcmQPq5BM0j8uCIg6/VsubP3sT82TcXKzyj2pdbcL+F5CK/YCExTUpmQIDAQAB`
	remotePrivateKeyStr = `MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCK7vS46B+uaCjj0IROr5mz5VOX/cihImUVSegJPn4rY4fQvFquQsp0ZRqS/KkTope2r5022n85lQAkEmnhoVetz1lp8JCan6KgfJjarXWq7dwgelFEMIJlKYI8AKB7invMp3pOZaYIp9cQX4EQ7C8dzlu560PqknGGkJlChNGBtn1PMelbmcio1/BNk61s5yoqKqpOiNDTCgw/uVj/aQT9RwE4Shk5bdlUJBYLsf6dBdIdW7kh5mQZR7Rp3uKHl/CTvCMGcxzQogQRiXIuDrvZdjc6Z3U7QNyZA+rkEzSPy4IiDr9Wy5s/exPzZNxcrPKPal1twv4XkIr9gITFNSmZAgMBAAECggEAe5qwerl5Ri9BAb2HmIG920DRqE2O61ywVcpU97Rzh6KbOGf6vUKK4Bb7F35V5jQnic6frieiPNaFM5J2RNjIKshookn2PLp9cw6m5xafsxy/VN29407NE7dkOIHORVslXSQ7OfhRSS4ZtmVhbG7UVE49aSEYYV88RR3sXDPSOPWSK7z6TVjEODdPhehbDxVCg/tGlYw9lt6eVlxSVgE5tDZf0RSnwjeoMwikP+1JV5Mxm+SkRxHNOnHr1HQrYeNksf29CcHVK+Wre63Np7oeGVvgJunecygPQeXIA0cc68+Z6c1wI1k8MXLk8up632FU+2hIxDXCPTVkC2hKxfEfyQKBgQDWGhHuzMU35lH+UdAuf4BIOXYdOgqf3cUZVmID6vER5w55McOdzj903DPU5H+5fFCyj7tMzRhRWHCIPbTLTueyKRyCQg1oeRMjgyEltFLW6FL5jVLhXU5J0a0b6B+qDkQ2N2oyCpuUp/sC61+Dy34KHMdXZpuMVU0QdamaC7gPowKBgQCmHyYv2ZbaSi/a3ac7xVBlQi2/Xd/Q9xzLsKn8vTPMq9gl7HXhwGon5rFbhU6gv2Qp5I0cyHrp1x9PMwXH6usRIB8Xmyyayi0ZwGA7T2sHSalgdIdr86BRR4p4FIiuWNV38IvwNwuKqK4vFyUMF33eEuphGKBgvIxyBKDk0FIFkwKBgQDT6bvkM9Pkr1hqs8mtrE9prU5GQWOwtk3W9VRQcmOnh54gwOvQrwrJ/QZkasIs8mnhQzhtHPc71KCViRYAwZm9Esn/96bTyDr0RF8ztZbk1dEC5impnLPXhuyjmY51wGctjo3S+ALkEZv2WMgSaADZu4Bm9s1xCiEb8IotSfolpwKBgQCc66Gz45N3QkrwMR7u/BVUgW4bbf6lMziVRJ1ebA9JUC7OrA4yoQLmDCoPLN64Q/LHC+ksfkh1KcuekbDtRwCj3bbhIqjA0yhFQg7lF8EfUjrYLVta4vjWroCjq6ntH2cOdECMOkMByRM40mEhifNQ2odiDtQ4bQMyFSMy4YIJVwKBgFpgNsYCvqInPgtYkup+SqIejFibyN2Vl+qIyLuhoYxBIQdtZGiCSAuhuBa8+dPToZfdEi5oFS7zjzZe+r4FiMIDuBPnWsDnLtaRdJG08cdRCCD+e1ck8CbTfveqks/+8zVZM1N0ck7i+lnZNNLciIyY5BrhvfAdeS1w9PA++DUh`
)

type FFRsa struct {
	localPubKey, remotePubKey *rsa.PublicKey
	localPriKey, remotePriKey *rsa.PrivateKey
}

var once sync.Once
var instance *FFRsa

func GetInstance() *FFRsa{
	once.Do(func() {
		instance = &FFRsa{}
		var err error
		instance.localPubKey, err = loadPublicKeyBase64(publicKeyStr)
		if  err != nil{
			log.LOGGER("APP").Error(err)
		}
		instance.remotePubKey, err = loadPublicKeyBase64(remotePublickeyStr)
		if  err != nil{
			log.LOGGER("APP").Error(err)
		}
		instance.localPriKey, err = loadPrivateKeyBase64(privateKeyStr)
		if  err != nil{
			log.LOGGER("APP").Error(err)
		}
		//instance.localPubKey = &instance.localPriKey.PublicKey

		instance.remotePriKey, err = loadPrivateKeyBase64(remotePrivateKeyStr)
		if  err != nil{
			log.LOGGER("APP").Error(err)
		}
		//instance.remotePubKey = &instance.remotePriKey.PublicKey
	})
	return instance
}

//注意可能需要采用分段加密的方式

func (f *FFRsa) EncodeWithLocalPubKey(ciphertext string) (string, error) {
	retStr, err := f.RsaEncryptBlock(ciphertext, f.localPubKey)
	//decodedtext := base64.StdEncoding.EncodeToString(retData)
	return retStr, err
}

func (f *FFRsa) DecodeWithLocalPrivateKey(ciphertext string) (string, error) {
	retData, err := f.RsaDecryptBlock(ciphertext, f.localPriKey)
	return string(retData), err
}


func (f *FFRsa) EncodeWithRemotePubKey(ciphertext string) (string, error) {
	retStr, err := f.RsaEncryptBlock(ciphertext, f.remotePubKey)
	//decodedtext := base64.StdEncoding.EncodeToString(retData)
	return retStr, err
}

func (f *FFRsa) DecodeWithRemotePrivateKey(ciphertext string) (string, error) {
	retData, err := f.RsaDecryptBlock(ciphertext, f.remotePriKey)
	return string(retData), err
}

//每隔多少个分割一下
func split(buf []byte, lim int) [][]byte {
	var chunk []byte
	chunks := make([][]byte, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:len(buf)])
	}
	return chunks
}

// 公钥加密
func (f *FFRsa) RsaEncryptBlock(data string, publicKey *rsa.PublicKey) (string, error) {

	partLen := f.localPubKey.N.BitLen() / 8 - 11
	chunks := split([]byte(data), partLen)
	buffer := bytes.NewBufferString("")
	for _, chunk := range chunks {
		bytes, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, chunk)
		if err != nil {
			return "", err
		}
		buffer.Write(bytes)
	}

	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

// 私钥解密
func (f *FFRsa) RsaDecryptBlock(encrypted string, privateKey *rsa.PrivateKey) (string, error) {
	partLen := f.localPubKey.N.BitLen() / 8
	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	chunks := split([]byte(raw), partLen)
	buffer := bytes.NewBufferString("")
	for _, chunk := range chunks {
		decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, chunk)
		if err != nil {
			return "", err
		}
		buffer.Write(decrypted)
	}
	return buffer.String(), err
}

// Load private key from base64
func loadPrivateKeyBase64(base64key string) (*rsa.PrivateKey, error) {
	keybytes, err := base64.StdEncoding.DecodeString(base64key)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed, error=%s\n", err.Error())
	}
	privatekey, err := x509.ParsePKCS8PrivateKey(keybytes)
	if err != nil {
		return nil, err
	}
	return privatekey.(*rsa.PrivateKey), nil
}

//加载公钥字符串获取公钥对象
func loadPublicKeyBase64(base64key string) (*rsa.PublicKey, error) {

	keybytes, err := base64.StdEncoding.DecodeString(base64key)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed, error=%s\n", err.Error())
	}
	pubkeyinterface, err := x509.ParsePKIXPublicKey(keybytes)
	if err != nil {
		return nil, err
	}
	publickey := pubkeyinterface.(*rsa.PublicKey)
	return publickey, nil
}

