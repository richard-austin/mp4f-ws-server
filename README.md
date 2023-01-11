# mp4f-ws-server
An MP4f server for live streaming from an RTSP service to a WSE browser front end

Under development:-

#### Stage 1.0.0: 
* Simple proxy, RTSP to MP4f


#### Stage 1.0.1:
* Handle multiple clients.
* Save ftyp and moov atoms sent from ffmpeg after connection and send those to cients ahead of the ongoing live data.
* Get the codec info (video and, optionally, audio) from the moov trak boxes and save against the corresponding stream id. 

#### Stage 1.0.2
* HTTP service at http://\<base address\>/h which does not send the codec information first, just ftyp then moov before the main stream. This access point is used for ffmpeg to connect to when making recordings.
