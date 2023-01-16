package main

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"os"
)

type StreamC struct {
	Descr        string `json:"descr"`
	AudioBitRate string `json:"audio_bitrate"`
	NetcamUri    string `json:"netcam_uri"`
	ClientUri    string `json:"client_uri"`
	URI          string `json:"uri"`
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
	data, err := os.ReadFile("src/cameras.json")
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(data, &tmp.Cameras)
	if err != nil {
		log.Fatalln(err)
	}
	return &tmp
}
