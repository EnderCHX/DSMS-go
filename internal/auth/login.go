package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/go-resty/resty/v2"
)

var authApi = "https://api.passport.hrbeu.top"

/*
return refresh_token, access_token, err
*/
func Login(username, password string) (string, string, error) {

	h := sha256.New()
	h.Write([]byte(password))
	password = hex.EncodeToString(h.Sum(nil))

	client := resty.New()
	req, err := client.R().
		SetBody(map[string]string{
			"username": username,
			"password": password,
		}).
		Post(authApi + "/login")

	if err != nil {
		return "", "", err
	}

	resCode, _ := sonic.Get(req.Body(), "code")
	resMsg, _ := sonic.Get(req.Body(), "msg")
	resMsgStr, _ := resMsg.String()

	if resCodeStr, _ := resCode.String(); resCodeStr != "Success" {
		return "", "", fmt.Errorf("%v", resMsgStr)
	}

	reqData, _ := sonic.Get(req.Body(), "data")
	refresh_token, _ := reqData.Get("refresh_token").String()
	access_token, _ := reqData.Get("access_token").String()

	return refresh_token, access_token, nil
}
