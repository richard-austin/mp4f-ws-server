package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"time"
)

func main() {

	maxInstances := 12

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for stream := 1; stream < 9; stream++ {
		for instance := 1; instance < maxInstances+1; instance++ {

			go func(stream int, instance int) {
				for {
					cmdStr := fmt.Sprintf("/usr/bin/ffmpeg -y -f mp4 -i https://192.168.0.29/h/stream?suuid=stream%d -c copy -f mp4 test_file_%d_%d.mp4f", stream, stream, instance)
					cmd := exec.Command("bash", "-c", cmdStr)
					_, err := cmd.Output()
					if err != nil {
						fmt.Printf("%s", err.Error())
					} else {
						fmt.Printf("%s", "OK")
					}
					time.Sleep(time.Second)
				}
			}(stream, instance)
		}
	}
	for range c {
		return
		// sig is a ^C, handle it
	}
}
