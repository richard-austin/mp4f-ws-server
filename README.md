# mp4f-ws-server
##  An MP4f server for live streaming from an RTSP service to an MSE browser front end
### Features
* Stream RTSP sources to the web
* mp4f streaming for low latency
* ffmpeg links this server with RTSP sources, giving flexiblity in source format and transcoding
* Basic web page to check the streams
* Stream url with codecs preamble for easy etup of MSE clients
* Stream url without codecs preamle for other clients (such as ffmpeg)

### Configuration
The configuration file is cameras.json, as shown below. There is a section for each camera with one or more streams (RTSP sources) supported for each camera.

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
        "uri": "http://localhost:8081/ws/stream/suuid=stream1",
        "video_height": 1440,
        "video_width": 2560
      },
      "stream2": {
        "audio_bitrate": "0",
        "descr": "SD",
        "netcam_uri": "rtsp://192.168.0.23:554/12",
        "client_uri":  "http://localhost:8081/live/stream?suuid=stream2",
        "uri": "http://localhost:8081/ws/stream/suuid=stream2",
        "video_height": 448,
        "video_width": 800
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
        "uri": "http://localhost:8081/ws/stream/suuid=stream3",
        "video_height": 1080,
        "video_width": 1920
      },
      "stream4": {
        "audio_bitrate": "0",
         "descr": "SD",
        "netcam_uri": "rtsp://192.168.0.26:554/12",
        "client_uri": "http://localhost:8081/live/stream?suuid=stream4",
        "uri": "http://localhost:8081/ws/stream/suuid=stream4",
        "video_height": 352,
        "video_width": 640
      }
    }
  }
}

```
The parameters in cameras.json are as described below.
#### Parameters
* **name** The camera name (its location). Th
