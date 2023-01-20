package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	minDelay := 3
	maxDelay := 200

	maxInstances := 10
	autoKill := true
	autoRestart := false
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for stream := 1; stream < 9; stream++ {
		for instance := 1; instance < maxInstances+1; instance++ {

			go func(stream int, instance int) {
				for {
					cmdStr := fmt.Sprintf("/usr/bin/ffmpeg -y -f mp4 -i https://192.168.0.29/h/stream?suuid=stream%d -c copy -f mp4 test_file_%d_%d.mp4f", stream, stream, instance)
					cmd := exec.Command("bash", "-c", cmdStr)
					cmd.Start()
					pid := cmd.Process.Pid

					if autoKill {
						go func() {
							delay := rand.Intn(maxDelay-minDelay+1) + minDelay
							time.Sleep(time.Second * time.Duration(delay))
							syscall.Kill(pid, syscall.SIGINT)
						}()
					}
					err := cmd.Wait()
					ee := err.(*exec.ExitError)
					if ee != nil && ee.ExitCode() != 255 {
						log.Errorf("ffmpeg :- %s, %s", string(ee.Stderr), ee.Error())
					} else {
						log.Errorf("ffmpeg :- :- %s", err.Error())
					}
					if autoRestart {
						time.Sleep(time.Second)
					} else {
						break
					}
				}
			}(stream, instance)
		}
	}
	for range c {
		return
		// sig is a ^C, handle it
	}
}
