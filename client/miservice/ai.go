package miservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type AIService struct {
	account *Account
}

func NewAIService(account *Account) *AIService {
	return &AIService{
		account: account,
	}
}

func (mnas *AIService) Request(uri string, data url.Values, out any) error {
	requestId := "app_ios_" + getRandom(30)
	if data != nil {
		data["requestId"] = []string{requestId}
	} else {
		uri += "&requestId=" + requestId
	}

	headers := http.Header{
		"User-Agent": []string{"MiHome/6.0.103 (com.xiaomi.mihome; build:6.0.103.1; iOS 14.4.0) Alamofire/6.0.103 MICO/iOSApp/appStore/6.0.103"},
	}

	return mnas.account.Request("micoapi", "https://api2.mina.mi.com"+uri, data, nil, headers, true, out)
}

type DeviceData struct {
	DeviceID     string `json:"deviceID"`
	SerialNumber string `json:"serialNumber"`
	Name         string `json:"name"`
	Alias        string `json:"alias"`
	Current      bool   `json:"current"`
	Presence     string `json:"presence"`
	Address      string `json:"address"`
	MiotDID      string `json:"miotDID"`
	Hardware     string `json:"hardware"`
	RomVersion   string `json:"romVersion"`
	Capabilities struct {
		ChinaMobileIms      int `json:"china_mobile_ims"`
		SchoolTimetable     int `json:"school_timetable"`
		NightMode           int `json:"night_mode"`
		UserNickName        int `json:"user_nick_name"`
		PlayerPauseTimer    int `json:"player_pause_timer"`
		DialogH5            int `json:"dialog_h5"`
		ChildMode2          int `json:"child_mode_2"`
		ReportTimes         int `json:"report_times"`
		AlarmVolume         int `json:"alarm_volume"`
		AiInstruction       int `json:"ai_instruction"`
		ClassifiedAlarm     int `json:"classified_alarm"`
		AiProtocol30        int `json:"ai_protocol_3_0"`
		NightModeDetail     int `json:"night_mode_detail"`
		ChildMode           int `json:"child_mode"`
		BabySchedule        int `json:"baby_schedule"`
		ToneSetting         int `json:"tone_setting"`
		Earthquake          int `json:"earthquake"`
		AlarmRepeatOptionV2 int `json:"alarm_repeat_option_v2"`
		XiaomiVoip          int `json:"xiaomi_voip"`
		NearbyWakeupCloud   int `json:"nearby_wakeup_cloud"`
		FamilyVoice         int `json:"family_voice"`
		BluetoothOptionV2   int `json:"bluetooth_option_v2"`
		Yunduantts          int `json:"yunduantts"`
		MicoCurrent         int `json:"mico_current"`
		VoipUsedTime        int `json:"voip_used_time"`
	} `json:"capabilities"`
	RemoteCtrlType  string `json:"remoteCtrlType"`
	DeviceSNProfile string `json:"deviceSNProfile"`
	DeviceProfile   string `json:"deviceProfile"`
	BrokerEndpoint  string `json:"brokerEndpoint"`
	BrokerIndex     int    `json:"brokerIndex"`
	Mac             string `json:"mac"`
	Ssid            string `json:"ssid"`
}

type Devices struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Data    []DeviceData `json:"data"`
}

func (mnas *AIService) DeviceList(master int) (devices []DeviceData, err error) {
	var res Devices
	err = mnas.Request(fmt.Sprintf("/admin/v2/device_list?master=%d", master), nil, &res)
	if err != nil {
		return nil, err
	}

	return res.Data, nil
}

func (mnas *AIService) UbusRequest(deviceId, method, path string, message map[string]interface{}, res any) error {
	messageJSON, _ := json.Marshal(message)
	data := url.Values{
		"deviceId": []string{deviceId},
		"message":  []string{string(messageJSON)},
		"method":   []string{method},
		"path":     []string{path},
	}

	err := mnas.Request("/remote/ubus", data, res)
	if err != nil {
		return err
	}
	return nil
}

func (mnas *AIService) TextToSpeech(deviceId, text string) (map[string]interface{}, error) {
	var res map[string]interface{}
	err := mnas.UbusRequest(deviceId, "text_to_speech", "mibrain", map[string]interface{}{"text": text}, &res)
	return res, err
}

func (mnas *AIService) PlayerSetVolume(deviceId string, volume int) (map[string]interface{}, error) {
	var res map[string]interface{}
	err := mnas.UbusRequest(deviceId, "player_set_volume", "mediaplayer", map[string]interface{}{"volume": volume, "media": "app_ios"}, &res)
	return res, err
}

func (mnas *AIService) PlayerPause(deviceId string) (map[string]interface{}, error) {
	var res map[string]interface{}
	err := mnas.UbusRequest(deviceId, "player_play_operation", "mediaplayer", map[string]interface{}{"action": "pause", "media": "app_ios"}, &res)
	return res, err
}

func (mnas *AIService) PlayerStop(deviceId string) (map[string]interface{}, error) {
	var res map[string]interface{}
	err := mnas.UbusRequest(deviceId, "player_play_operation", "mediaplayer", map[string]interface{}{"action": "stop", "media": "app_ios"}, &res)
	return res, err
}

func (mnas *AIService) PlayerPlay(deviceId string) (map[string]interface{}, error) {
	var res map[string]interface{}
	err := mnas.UbusRequest(deviceId, "player_play_operation", "mediaplayer", map[string]interface{}{"action": "play", "media": "app_ios"}, &res)
	return res, err
}

type PlayerStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Code int    `json:"code"` //noError=0
		Info string `json:"info"`
	} `json:"data"`
}
type InfoData struct {
	Status         int `json:"status"`
	Volume         int `json:"volume"`
	LoopType       int `json:"loop_type"`
	MediaType      int `json:"media_type"`
	PlaySongDetail struct {
		AudioID  string `json:"audio_id"`
		Position int    `json:"position"`
		Duration int    `json:"duration"`
	} `json:"play_song_detail"`
	TrackList []string `json:"track_list"`
}

func (mnas *AIService) PlayerGetStatus(deviceId string) (*PlayerStatus, error) {
	var res PlayerStatus
	err := mnas.UbusRequest(deviceId, "player_get_play_status", "mediaplayer", map[string]interface{}{"media": "app_ios"}, &res)
	return &res, err
}

/*async def player_set_loop(self, deviceId, type=1): //"loop"=0
  return await self.ubus_request(
      deviceId,
      "player_set_loop",
      "mediaplayer",
      {"media": "common", "type": type},
  )
*/
func (mnas *AIService) PlayerSetLoop(deviceId string, playType int) (map[string]interface{}, error) {
	var res map[string]interface{}
	err := mnas.UbusRequest(deviceId, "player_set_loop", "mediaplayer", map[string]interface{}{"media": "common", "type": playType}, &res)
	return res, err
}

func (mnas *AIService) PlayByUrl(deviceId, url string) (map[string]interface{}, error) {
	var res map[string]interface{}
	err := mnas.UbusRequest(deviceId, "player_play_url", "mediaplayer", map[string]interface{}{"url": url, "type": 1, "media": "app_ios"}, &res)
	return res, err
}

// 声明Music相关结构体（具名类型，可复用）
type Music struct {
	Payload      MusicPayload `json:"payload"`
	PlayBehavior string       `json:"play_behavior"`
}

type MusicPayload struct {
	AudioType  string      `json:"audio_type"`
	AudioItems []AudioItem `json:"audio_items"`
	ListParams ListParams  `json:"list_params"`
}

type AudioItem struct {
	ItemId ItemID      `json:"item_id"`
	Stream AudioStream `json:"stream"`
}

type ItemID struct {
	AudioId string `json:"audio_id"`
	Cp      CpInfo `json:"cp"`
}

type CpInfo struct {
	AlbumId      string `json:"album_id"`
	EpisodeIndex int    `json:"episode_index"`
	Id           string `json:"id"`
	Name         string `json:"name"`
}

type AudioStream struct {
	Url string `json:"url"`
}

type ListParams struct {
	ListId         string `json:"listId"`
	LoadmoreOffset int    `json:"loadmore_offset"`
	Origin         string `json:"origin"`
	Type           string `json:"type"`
}

func (mnas *AIService) PlayByMusicUrl(deviceId, url string) (map[string]interface{}, error) {
	audio_id := "1582971365183456177"
	id := "355454500"
	audio_type := "MUSIC" // "",If set to "MUSIC", the light will be on
	music := Music{
		Payload: MusicPayload{
			AudioType: audio_type,
			AudioItems: []AudioItem{
				{
					ItemId: ItemID{
						AudioId: audio_id,
						Cp: CpInfo{
							AlbumId:      "-1",
							EpisodeIndex: 0,
							Id:           id,
							Name:         "xiaowei",
						},
					},
					Stream: AudioStream{
						Url: url,
					},
				},
			},
			ListParams: ListParams{
				ListId:         "-1",
				LoadmoreOffset: 0,
				Origin:         "xiaowei",
				Type:           "MUSIC",
			},
		},
		PlayBehavior: "REPLACE_ALL",
	}
	jsonBytes, _ := json.Marshal(music)
	var res map[string]interface{}
	err := mnas.UbusRequest(deviceId, "player_play_music", "mediaplayer", map[string]interface{}{"startaudioid": audio_id, "music": string(jsonBytes)}, &res)
	return res, err
}

func (mnas *AIService) SendMessage(devices []DeviceData, devno int, message string, volume *int) (bool, error) {
	result := false
	for i, device := range devices {
		if devno == -1 || devno != i+1 || device.Capabilities.Yunduantts != 0 /*nil*/ {
			deviceId := device.DeviceID
			if volume != nil {
				res, err := mnas.PlayerSetVolume(deviceId, *volume)
				result = err == nil && res != nil
			}
			if message != "" {
				res, err := mnas.TextToSpeech(deviceId, message)
				result = err == nil && res != nil
			}
			if devno != -1 || !result {
				break
			}
		}
	}

	return result, nil
}
