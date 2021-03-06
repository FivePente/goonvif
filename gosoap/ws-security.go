package gosoap

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"io"
	"math/rand"
	"time"

	"github.com/elgs/gostrgen"
)

/*************************
	WS-Security types
*************************/
const (
	passwordType      = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest"
	encodingType      = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary"
	wssecuritySecext  = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
	wssecurityUtility = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

//Security type :XMLName xml.Name `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
type Security struct {
	//XMLName xml.Name  `xml:"wsse:Security"`
	XMLName       xml.Name      `xml:"wsse:Security"`
	UsernameToken usernameToken `xml:"wsse:UsernameToken"`
	Wsse          string        `xml:"xmlns:wsse,attr"`
	Wsu           string        `xml:"xmlns:wsu,attr"`
}

type password struct {
	//XMLName xml.Name `xml:"wsse:Password"`
	Type     string `xml:"Type,attr"`
	Password string `xml:",chardata"`
}

type nonce struct {
	//XMLName xml.Name `xml:"wsse:Nonce"`
	//Type  string `xml:"Type,attr"`
	Nonce string `xml:",chardata"`
}

type usernameToken struct {
	//	XMLName  xml.Name `xml:"wsse:UsernameToken"`
	Username string   `xml:"wsse:Username"`
	Password password `xml:"wsse:Password"`
	Nonce    nonce    `xml:"wsse:Nonce"`
	Created  string   `xml:"wsu:Created"`
}

/*
   <Security s:mustUnderstand="1" xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd">
       <UsernameToken>
           <Username>admin</Username>
           <Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">edBuG+qVavQKLoWuGWQdPab4IBE=</Password>
           <Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">S7wO1ZFTh0KXv2CR7bd2ZXkLAAAAAA==</Nonce>
           <Created xmlns="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">2018-04-10T18:04:25.836Z</Created>
       </UsernameToken>
   </Security>
*/

//NewSecurity get a new security
func NewSecurity(username, passwd string) Security {
	/** Generating Nonce sequence **/
	charsToGenerate := 32
	charSet := gostrgen.Lower | gostrgen.Digit

	nonceSeq, _ := gostrgen.RandGen(charsToGenerate, charSet, "", "")

	//use the same time for password and Created fileld !
	nowTimeUTC := time.Now().UTC()

	return Security{
		UsernameToken: usernameToken{
			Username: username,
			Password: password{
				Type:     passwordType,
				Password: generateToken(username, nonceSeq, nowTimeUTC, passwd),
			},
			Nonce: nonce{
				Nonce: nonceSeq,
			},
			Created: nowTimeUTC.Format(time.RFC3339Nano),
		},
		Wsse: wssecuritySecext,
		Wsu:  wssecurityUtility,
	}

}

//Digest = B64ENCODE( SHA1( B64DECODE( Nonce ) + Date + Password ) )
func generateToken(Username string, Nonce string, Created time.Time, Password string) string {

	sDec, _ := base64.StdEncoding.DecodeString(Nonce)

	hasher := sha1.New()
	//hasher.Write([]byte((base64.StdEncoding.EncodeToString([]byte(Nonce)) + Created.Format(time.RFC3339) + Password)))
	hasher.Write([]byte(string(sDec) + Created.Format(time.RFC3339Nano) + Password))

	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}

func generatePasswdDigest(nonce, date, passwd string) string {
	h := sha1.New()
	io.WriteString(h, nonce)
	io.WriteString(h, date)
	io.WriteString(h, passwd)
	res := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(res)
}

//randStringBytesMaskImprSrc random bytes
//nonce := RandStringBytesMaskImprSrc(8)
func randStringBytesMaskImprSrc(n int) []byte {
	nonce := make([]byte, n)
	srcRand := rand.NewSource(time.Now().UnixNano())
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, srcRand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = srcRand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			nonce[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return nonce
}
