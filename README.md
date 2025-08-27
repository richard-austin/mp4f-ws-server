```txt
It turns out that the -r option only works for raw stream data. (This can be an H.264 stream... it does not have to be just pixel data.)

In this example, I'm using MP4 containers. First, extract the stream:

ffmpeg -i source.mp4 -map 0:v -vcodec copy -bsf:v h264_mp4toannexb source-video.h264
Next, take that stream and re-mux it, but with a specific frame rate and generated timestamps.

ffmpeg -fflags +genpts -r 60 -i source-video.h264 -vcodec copy output.mp4
```


# RTSPToFmp4
##  A fragmented MP4 media server for live streaming from an RTSP service to an MSE browser front end
### Features
* Stream RTSP sources to the web
* fmp4 streaming for low latency
* ffmpeg links this server with RTSP sources, giving flexiblity in source format and transcoding
* Basic web page to check the streams
* Stream url with codecs preamble for easy setup of MSE clients
* Stream url without codecs preamble for other clients (such as ffmpeg)

### Configuration
The configuration files are cameras.json to specify the camera streams and config.json for general server configuration.

#### cameras.json
```json
{
  "camera1": {
    "name": "Doorbell",
    "username": "user",
    "password": "password",
    "rtsp_transport": "udp",
    "streams": {
      "stream1": {
        "audio": true,
        "audio_encoding": "aac",
        "descr": "HD",
        "netcam_uri": "rtsp://192.168.1.43:554",
        "media_server_input_uri": "http://localhost:8081/live/stream?suuid=stream1",
        "uri": "http://localhost:8081/ws/stream/suuid=stream1"
      }
    }
  },
  "camera2": {
    "name": "Garden PTZ",
    "username": "",
    "password": "",
    "rtsp_transport": "tcp",
    "streams": {
      "stream2": {
        "audio": true,
        "audio_encoding": "ulaw",
        "descr": "HD",
        "netcam_uri": "rtsp://192.168.1.30:554/11",
        "media_server_input_uri": "http://localhost:8081/live/stream?suuid=stream2",
        "uri": "http://localhost:8081/ws/stream/suuid=stream2"
      },
      "stream3": {
        "audio": true,
        "audio_encoding": "ulaw",
        "descr": "SD",
        "netcam_uri": "rtsp://192.168.1.30:554/12",
        "media_server_input_uri": "http://localhost:8081/live/stream?suuid=stream3",
        "uri": "http://localhost:8081/ws/stream/suuid=stream3"
      }
    }
  }
}

```
The parameters in cameras.json are as described below.
#### Parameters
* **camera**(*n*)
    * **name** The camera name. This followed by the the relevant stream description is the descriptive text on the stream selector buttons on the test web page.
    * **username** The username idf required for authentication. If not required, set to empty string
    * **password** The password if required for authentication. If not required, set to empty string
    *  **rtsp_transport** The RTSP transport used by ffmpeg, may be tcp or udp
    * **streams**
        * **stream**(*n*)
            * **audio** Set to true to use cameras rtsp audio stream, else set to false if not supported or to ignore the audio.
            * **audio_encoding** If the cameras audio format is AAC, set to aac, so ffmpeg will use copy for the audio. Setting to anything else will cause ffmpeg to encode the audio to AAC.
            *  **descr** Description of stream (say HD or SD). This follows the camera name for the descriptive text on the stream selector buttons on the test web page.
            *  **netcam_uri** The URL of this stream from the net camera.
            *  **client_uri** The URL ffmpeg must use to connect to the server input side. This is generally of the form [http://localhost:8081/live/stream?suuid=stream(*n*)]
            *  **uri** The websocket URL which MSE connects to. This is generally of the form [http://my-IP-address:8081/ws/stream?suuid=stream(*n*)]

#### config.json
```json
{
  "log_path": "/var/log/mp4f-server/mp4f-server.log",
  "log_level": "INFO",
  "server_port": 8081,
  "gop_cache": true
}
```
The parameters in config.json are as described below.
#### Parameters
* **log_path** The path where the log files will be written. The ffmpeg logs will also be written to the path as the server log. 
* **log_level** The required level of logging (can be "PANIC", "FATAL", "ERROR", "WARN", "INFO", "DEBUG", or "TRACE")
* **server_port** The port the server will listen on (for web page, ffmpeg input and websocket output)
* **gop_cache** Enables the GOP cache if true. Without the GOP cache, the server will wait for the next keyframe to start the video stream with, resulting in an initial delay. When set to true, the video will start immediately.

### Setting up
#### ffmpeg is required, If not already installed
_Note that ffmpeg version 5 and above do not work correctly with RTSP streams which include audio.
The raw H264/H265 do not have the timestamps those versions require to sync with the audio.
So far I've not found any parameters which fix the problem with versions 5 and 6._

```
sudo apt install ffmpeg
(Or as appropriate for your OS)
```
#### Download repo and set up log directory
```
git clone git@github.com:richard-austin/mp4f-ws-server.git
sudo mkdir /var/log/mp4f-server
sudo chown your-user:your-user /var/log/mp4f-server
```

#### Run
```
cd mp4f-ws-server
GO111MODULE=on go run src/*.go
```
#### Build and run
```
cd mp4f-ws-server/src
go build -o mp4f-ws-server
cd ..
src/mp4f-ws-server
```
### Build for arm64 e.g. Raspberry pi
An executable file for arm64 processors can be built on an x86 devlopment system
```
env GOOS=linux GOARCH=arm64 go build -o mp4f-ws_server_arm64
```
### View the streams on the web page
set browser to http://localhost:8081
### Websocket URL for mse
ws://localhost:8081/ws/stream?suuid=stream1 (for stream1)
### General (no codec header) http URL
http://localhost:8081/h/stream?suuid=stream1 (for stream1)
### Record stream1 with ffmpeg
ffmpeg -f mp4 -i http://localhost:8081/h/stream?suuid=stream1 -f mp4 output.mp4f

go version go1.18.1 linux/amd64

