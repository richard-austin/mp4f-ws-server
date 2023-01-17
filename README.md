# mp4f-ws-server
##  An MP4f server for live streaming from an RTSP service to an MSE browser front end
### Features
* Stream RTSP sources to the web
* mp4f streaming for low latency
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
    "name": "Garden PTZ",
    "streams": {
      "stream1": {
        "audio_bitrate": "16000",
        "descr": "HD",
         "netcam_uri": "rtsp://192.168.0.23:554/11",
        "client_uri": "http://localhost:8081/live/stream?suuid=stream1",
        "uri": "http://localhost:8081/ws/stream/suuid=stream1"
      },
      "stream2": {
        "audio_bitrate": "0",
        "descr": "SD",
        "netcam_uri": "rtsp://192.168.0.23:554/12",
        "client_uri":  "http://localhost:8081/live/stream?suuid=stream2",
        "uri": "http://localhost:8081/ws/stream/suuid=stream2"
      }
    }
  },
  "camera2": {
    "name": "Porch",
    "streams": {
      "stream3": {
        "audio_bitrate": "0",
        "descr": "HD",
        "netcam_uri": "rtsp://192.168.0.26:554/11",
        "client_uri":  "http://localhost:8081/live/stream?suuid=stream3",
        "uri": "http://localhost:8081/ws/stream/suuid=stream3"
      },
      "stream4": {
        "audio_bitrate": "0",
         "descr": "SD",
        "netcam_uri": "rtsp://192.168.0.26:554/12",
        "client_uri": "http://localhost:8081/live/stream?suuid=stream4",
        "uri": "http://localhost:8081/ws/stream/suuid=stream4"
      }
    }
  }
}

```
The parameters in cameras.json are as described below.
#### Parameters
* **camera**(*n*)
    * **name** The camera name. This followed by the the relevant stream description is the descriptive text on the stream selector buttons on the test web page.
    * **streams**
        * **stream**(*n*)
            * **audio_bitrate** Audio resampling bitrate. As used in the -ar parameter of ffmpeg. Values can be 8000, 24000, 32000, 40000 or 48000. If the value 0 is used, audio will be disabed on the stream (-an on ffmpeg)
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
  "default_latency_limit": 0.8
}
```
The parameters in config.json are as described below.
#### Parameters
* **log_path** The path where the log files will be wriiten
* **log_level** The required level of logging (can be "PANIC", "FATAL", "ERROR", "WARN", "INFO", "DEBUG", or "TRACE")
* **server_port** The port the server will listen on (for web page, ffmpeg input and websocket output)
* **default_latency_limit** This is the initial value for the latency limit when the web page is initially loaded. The value in use can be changed dynamically on the web page with a selector. When a new stream is selected, it will revert to the value given here. The latency limit determines how far behind real time the video must run before it is pulled in to a shorter delay. If this value is too high the latency can get larger than you might want. If set too low, poor stability can result. The optimum value depends on the network quality and the data rate of the stream. 


### Setting up

```
git clone git@github.com:richard-austin/mp4f-ws-server.git
sudo mkdir /var/log/mp4f-server
sudo chown your-user:your-user /var/log/mp4f-server
cd mp4f-ws-server/src
go build -o mp4f-ws-server
cd ..
src/mp4f-ws-server
--or--
GO111MODULE=on go run src/*.go
```
### View the streams on the web page
set browser to http://localhost:8081

### Build for arm64 e.g. Raspberry pi
An executable file for arm64 processors can be built on an x86 devlopment system
```
env GOOS=linux GOARCH=arm64 go build -o mp4f-ws_server_arm64
```

go version go1.18.1 linux/amd64

