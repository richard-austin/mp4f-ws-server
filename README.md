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

## Setting up

