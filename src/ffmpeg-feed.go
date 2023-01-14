package main

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"time"
)

func ffmpegFeed() {
	go func() {
		for {
			time.Sleep(time.Second)
			cmd := exec.Command("bash", "-c", "/usr/bin/ffmpeg -hide_banner -loglevel error -stimeout 1000000 -fflags nobuffer -rtsp_transport tcp -i rtsp://192.168.0.55:554/11 -c:v copy -c:a aac -async 1 -movflags empty_moov+omit_tfhd_offset+frag_keyframe+default_base_moof -frag_size 10 -preset superfast -tune zerolatency -f mp4 http://localhost:8081/live/s1?suuid=stream1")
			stdout, err := cmd.Output()

			if err != nil {
				ee := err.(*exec.ExitError)
				if ee != nil {
					log.Errorf("ffmpeg error :- %s, %s", string(ee.Stderr), ee.Error())
				} else {
					log.Errorf("ffmpeg error :- %s", err.Error())
				}
			} else if stdout != nil {
				log.Infof("ffmpeg output:- %s ", string(stdout))
			}
		}
	}()
}
