package main

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

type StreamC struct {
	Descr           string `json:"descr"`
	AudioBitRate    string `json:"audio_bitrate"`
	AudioEncoding   string `json:"audio_encoding"`
	AudioSampleRate int    `json:"audio_sample_rate"`
	//   DefaultMultiDisplay bool `json:"defaultOnMultiDisplay"`
	//   "descr": "HD",
	//"motion": {
	//"enabled": false,
	//"mask_file": "",
	//"trigger_recording_on": ""
	//},
	NetcamUri string `json:"netcam_uri"`
	NMSUri    string `json:"nms_uri"`

	//	"recording": {
	//"enabled": true,
	//"location": "rec1",
	//"uri": "http://localhost:8084/recording/rec1/"
	//},
	URI         string `json:"uri"`
	VideoHeight int    `json:"video_height"`
	VideoWidth  int    `json:"video_width"`
}

type Camera struct {
	Name    string             `json:"name"`
	Address string             `json:"address"`
	Streams map[string]StreamC `json:"streams"`
}

type Cameras struct {
	Cameras map[string]Camera `json:"{}"`
}

func (c *Cameras) Suuids() (suuids map[string]string) {
	suuids = map[string]string{}
	for _, camera := range c.Cameras {
		for k, stream := range camera.Streams {
			suuids[camera.Name+" "+stream.Descr] = k
		}
	}
	return
}

func loadConfig() *Cameras {
	var tmp Cameras
	data, err := ioutil.ReadFile("src/cameras.json")
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(data, &tmp.Cameras)
	if err != nil {
		log.Fatalln(err)
	}
	return &tmp
}
