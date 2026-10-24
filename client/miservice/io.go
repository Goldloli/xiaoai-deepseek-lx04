package miservice

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type IOService struct {
	account *Account
	server  string
}

const MIIO_SID = "xiaomiio"

func NewIOService(account *Account, region *string) *IOService {
	server := "https://"
	if region != nil && *region != "cn" {
		server += *region + "."
	}
	server += "api.io.mi.com/app"
	return &IOService{account: account, server: server}
}

func (s *IOService) Request(uri string, data map[string]interface{}) (map[string]interface{}, error) {
	prepareData := func(token *Tokens, cookies map[string]string) url.Values {
		cookies["PassportDeviceId"] = token.DeviceId
		/*cookies["countryCode"] = "CN"
		  cookies["locale"] = "zh_CN"
		  cookies["timezone"] = "GMT+08:00"
		  cookies["timezone_id"] = "Asia/Shanghai"
		  cookies["userId"] = token.UserId
		  cookies["cUserId"] = account.pass?.cUserId
		  cookies["PassportDeviceId"] = token.DeviceId
		  cookies["serviceToken"] = account.serviceToken
		  cookies["yetAnotherServiceToken"] = account.serviceToken*/
		return signData(uri, data, token.Sids[MIIO_SID].Ssecurity)
	}

	headers := http.Header{
		"User-Agent":                 []string{"iOS-14.4-6.0.103-iPhone12,3--D7744744F7AF32F0544445285880DD63E47D9BE9-8816080-84A3F44E137B71AE-iPhone"},
		"x-xiaomi-protocal-flag-cli": []string{"PROTOCAL-HTTP2"},
		//"User-Agent":                 []string{"MICO/AndroidApp/@SHIP.TO.2A2FE0D7@/2.4.40"},
		//"miot-accept-encoding":       []string{"GZIP"},
		//"miot-encrypt-algorithm":     []string{"ENCRYPT-RC4"},
	}
	var resp map[string]interface{}
	err := s.account.Request(MIIO_SID, s.server+uri, nil, prepareData, headers, true, &resp)
	if err != nil {
		return nil, err
	}

	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error %s: %v", uri, resp)
	}

	return result, nil
}

/*
   {
     "adminFlag": 1,
     "bssid": "12:34:56:78:9A:BC",
     "desc": "Device online ",
     "did": "000000000",
     "extra": {
       "fw_version": "1.0.7",
       "isPasswordEncrypt": 0,
       "isSetPincode": 0,
       "needVerifyCode": 0,
       "pincodeType": 0
     },
     "family_id": 0,
     "internet_ip": "111.111.111.111",
     "isOnline": true,
     "latitude": "0.0",
     "localip": "192.168.3.1",
     "longitude": "0.0",
     "mac": "50:00:00:00:00:AA",
     "model": "xiaomi.wifispeaker.l05c",
     "name": "小爱音箱Play增强版",
     "p2p_id": "",
     "parent_id": "",
     "parent_model": "",
     "password": "",
     "pd_id": 4039,
     "permitLevel": 16,
     "pid": "0",
     "reset_flag": 0,
     "rssi": -51,
     "shareFlag": 0,
     "show_mode": 1,
     "ssid": "HUAWEI-H6",
     "token": "",
     "uid": 0000000000
   }
*/
type DeviceInfo struct {
	Name    string `json:"name"`
	Model   string `json:"model"`
	Did     string `json:"did"`
	LocalIP string `json:"localip"`
	Token   string `json:"token"`
}

func (s *IOService) DeviceList(getVirtualModel bool, getHuamiDevices int) (devices []DeviceInfo, err error) {
	data := map[string]interface{}{
		"getVirtualModel": getVirtualModel,
		"getHuamiDevices": getHuamiDevices,
	}
	result, err := s.Request("/home/device_list", data)
	if err != nil {
		return nil, err
	}
	deviceList := result["list"].([]interface{})

	devices = make([]DeviceInfo, len(deviceList))
	for i, item := range deviceList {
		device := item.(map[string]interface{})
		devices[i] = DeviceInfo{
			Name:    device["name"].(string),
			Model:   device["model"].(string),
			Did:     device["did"].(string),
			LocalIP: device["localip"].(string),
			Token:   device["token"].(string),
		}
	}
	return
}

func (s *IOService) HomeRequest(did, method string, params interface{}) (map[string]interface{}, error) {
	data := map[string]interface{}{
		"id":        1,
		"method":    method,
		"accessKey": "IOS00026747c5acafc2",
		"params":    params,
	}
	return s.Request("/home/rpc/"+did, data)
}

func (s *IOService) HomeGetProps(did string, props []string) (map[string]interface{}, error) {
	return s.HomeRequest(did, "get_prop", props)
}

func (s *IOService) HomeSetProps(did string, props map[string]interface{}) (map[string]int, error) {
	results := make(map[string]int, len(props))
	for prop, value := range props {
		result, err := s.HomeSetProp(did, prop, value)
		if err != nil {
			return nil, err
		}
		results[prop] = result
	}
	return results, nil
}

func (s *IOService) HomeGetProp(did, prop string) (interface{}, error) {
	results, err := s.HomeGetProps(did, []string{prop})
	if err != nil {
		return nil, err
	}
	return results[prop], nil
}

func (s *IOService) HomeSetProp(did, prop string, value interface{}) (int, error) {
	result, err := s.HomeRequest(did, "set_"+prop, value)
	if err != nil {
		return 0, err
	}
	if result["result"] == "ok" {
		return 0, nil
	}
	return -1, nil
}

//----------------- miot -----------------

func (s *IOService) MiotRequest(cmd string, params interface{}) (map[string]interface{}, error) {
	return s.Request("/miotspec/"+cmd, map[string]interface{}{"params": params})
}

type Iid struct {
	Siid int `json:"siid"`
	Piid int `json:"piid"`
}

func (s *IOService) MiotGetProps(did string, iids []Iid) ([]interface{}, error) {
	params := make([]map[string]interface{}, len(iids))
	for i, iid := range iids {
		params[i] = map[string]interface{}{
			"did":  did,
			"siid": iid.Siid,
			"piid": iid.Piid,
		}
	}
	result, err := s.MiotRequest("prop/get", params)
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(result))
	for i, it := range result {
		index, _ := strconv.Atoi(i)
		itm := it.(map[string]interface{})
		if code, ok := itm["code"].(int); ok && code == 0 {
			values[index] = itm["value"]
		} else {
			values[index] = nil
		}
	}
	return values, nil
}

func (s *IOService) MiotSetProps(did string, props map[Iid]interface{}) ([]int, error) {
	params := make([]map[string]interface{}, len(props))
	index := 0
	for i, prop := range props {
		params[index] = map[string]interface{}{
			"did":   did,
			"siid":  i.Siid,
			"piid":  i.Piid,
			"value": prop,
		}
		index++
	}
	result, err := s.MiotRequest("prop/set", params)
	if err != nil {
		return nil, err
	}

	codes := make([]int, len(result))
	for i, it := range result {
		index, _ := strconv.Atoi(i)
		itm := it.(map[string]interface{})
		codes[index] = itm["code"].(int)
	}
	return codes, nil
}

func (s *IOService) MiotGetProp(did string, iid Iid) (interface{}, error) {
	results, err := s.MiotGetProps(did, []Iid{iid})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (s *IOService) MiotSetProp(did string, iid Iid, value interface{}) (int, error) {
	results, err := s.MiotSetProps(did, map[Iid]interface{}{iid: value})
	if err != nil {
		return 0, err
	}
	return results[0], nil
}

func (s *IOService) MiotAction(did string, iid []int, args []interface{}) (float64, error) {
	result, err := s.MiotRequest("action", map[string]interface{}{
		"did":  did,
		"siid": iid[0],
		"aiid": iid[1],
		"in":   args,
	})
	if err != nil {
		return -1, err
	}
	return result["code"].(float64), nil
}
