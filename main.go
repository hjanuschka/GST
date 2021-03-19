package main

import (
	"fmt"
	"strings"
	"time"

	"os"

	"github.com/tinyzimmer/go-glib/glib"
	"github.com/tinyzimmer/go-gst/gst"
)

var playing bool

func main() {

	p := `
		compositor  async-handling=true  sync=true latency=5000000000  name=videomix !  tee name=vo 
		audiomixer  async-handling=true sync=true  latency=5000000000   name=audiomix async=true !   tee name=au

	

		hlssink2  async-handling=true name=output location=out/hlssink.%05d.ts playlist-location=out/out.m3u8  sync=true target-duration=10 playlist-length=5 max-files=5 name=hls 
		vo. 
				! queue ! videoconvert ! videoscale ! videorate
				!   x264enc cabac=false  key-int-max=50 quantizer=0  pass=0 speed-preset=ultrafast  tune=zerolatency bitrate=8000   byte-stream=true ! video/x-h264,width=1920,height=1080,pixel-aspect-ratio=1/1,framerate=25/1,format=RGBA,profile=constrained-baseline ! hls.video
		au. 
				! audioconvert
				! audiorate
				! audioresample
				! audio/x-raw, rate=44100, layout=interleaved, format=F32LE, channels=2 ! avenc_aac bitrate=128000 ac=2
				! audio/mpeg
                ! hls.audio


		audiotestsrc is-live=true wave=1  volume=0.1 !  audioconvert ! audioresample !  audio/x-raw,format=S32LE,rate=44100,channels=2 ! audiomix.
		videotestsrc is-live=true ! videoconvert !  video/x-raw,width=1920,height=1080,pixel-aspect-ratio=1/1,framerate=25/1,format=RGBA,profile=constrained-baseline ! videomix.
		

		
	
				`
	gst.Init(nil)

	mainLoop := glib.NewMainLoop(glib.MainContextDefault(), false)
	pipelineString := p

	pipeline, err := gst.NewPipelineFromString(pipelineString)
	if err != nil {
		fmt.Println("Pipeline Error")
		fmt.Println(err)
		os.Exit(2)
	}
	pipeline.SetState(gst.StatePlaying)
	playing = true
	go timeCodeUpdate(pipeline)

	pipeline.DebugBinToDotFile(gst.DebugGraphShowAll, "PLAYING")

	go func() {
		time.Sleep(time.Second * 10)
		AddNewUridecodeBin(pipeline)

	}()

	mainLoop.Run()
}

func AddNewUridecodeBin(pipeline *gst.Pipeline) *gst.Pipeline {

	// uridecodebin connection-speed=50000000  uri="https://HLS3"       sync=false async=true name=d2
	// d2. ! videoconvert ! videoscale ! video/x-raw,width=1920,height=1080 ! videomix.
	// d2. ! audioconvert
	//  ! audiorate
	//  ! audioresample
	//  ! volume volume=1.0
	//  ! audiomix.

	els, _ := gst.NewElementMany("uridecodebin", "videoconvert", "videoscale", "audioconvert", "audiorate", "audioresample", "volume")

	uridecodebin := els[0]
	uridecodebin.Set("name", "dynamic1")
	uridecodebin.Set("uri", "https://bitcdn-kronehit.bitmovin.com/v2/hls/chunklist_b3128000.m3u8")
	uridecodebin.Set("sync", false)
	uridecodebin.Set("async", true)
	uridecodebin.SetState(gst.StatePaused)

	pipeline.Add(uridecodebin)

	//alreadyRun := false
	uridecodebin.Connect("pad-added", func(_ *gst.Element, srcPad *gst.Pad) {
		var isAudio, isVideo bool
		caps := srcPad.GetCurrentCaps()
		for i := 0; i < caps.GetSize(); i++ {
			st := caps.GetStructureAt(i)
			if strings.HasPrefix(st.Name(), "audio/") {
				isAudio = true
			}
			if strings.HasPrefix(st.Name(), "video/") {
				isVideo = true
			}
		}
		if isVideo {
			//audio/x-raw, format=(string)F32LE, layout=(string)non-interleaved, rate=(int)48000, channels=(int)2, channel-mask=(bitmask)0x0000000000000003 in anything we support

			video, _ := gst.NewElementMany("videoconvert", "videoscale")
			pipeline.AddMany(video...)
			for _, e := range video {
				e.SyncStateWithParent()
			}
			videoconvert := video[0]
			videoscale := video[1]

			videomixer, err := pipeline.GetElementByName("videomix")
			if err != nil {
				panic(err)
			}
			videoconvertSink := videoconvert.GetStaticPad("sink")
			srcPad.Link(videoconvertSink)
			uridecodebin.Link(videoconvert)
			videoconvert.LinkFiltered(videoscale, gst.NewCapsFromString("video/x-raw,width=1920,height=1080"))

			videoscale.LinkFiltered(videomixer, gst.NewCapsFromString("video/x-raw,format=I420,width=1920,height=1080,interlace-mode=progressive,pixel-aspect-ratio=1/1"))

			fmt.Println("Video Added")
			pipeline.DebugBinToDotFile(gst.DebugGraphShowAll, "VIDEOWORKS")

		}
		if isAudio {

			audio, _ := gst.NewElementMany("audioconvert", "audiorate", "audioresample", "volume")
			pipeline.AddMany(audio...)
			for _, e := range audio {
				e.SyncStateWithParent()
			}
			audioconvert := audio[0]
			audiorate := audio[1]
			audioresample := audio[2]
			volume := audio[3]

			audiomixer, err := pipeline.GetElementByName("audiomix")
			if err != nil {
				panic(err)
			}

			audioconvertSink := audioconvert.GetStaticPad("sink")
			srcPad.Link(audioconvertSink)

			uridecodebin.Link(audioconvert)

			e1 := audioconvert.LinkFiltered(audiorate, gst.NewCapsFromString("audio/x-raw,format=S32LE,rate=44100,channels=2"))
			fmt.Println(e1)
			e1 = audiorate.Link(audioresample)
			fmt.Println(e1)
			e1 = audioresample.Link(volume)
			fmt.Println(e1)

			e2 := volume.LinkFiltered(audiomixer, gst.NewCapsFromString("audio/x-raw,format=S32LE,rate=44100,channels=2"))
			fmt.Println(e2)
			fmt.Println("Audio Added")
			//uridecodebin.SetState(gst.StatePlaying)
			for _, e := range audio {
				e.SyncStateWithParent()
			}
			uridecodebin.SyncStateWithParent()
			pipeline.DebugBinToDotFile(gst.DebugGraphShowAll, "NOWFROZEN")

		}

		return

	})

	return pipeline
}

func timeCodeUpdate(pipeline *gst.Pipeline) {
	for range time.NewTicker(time.Second).C {

		pos := gst.NewPositionQuery(gst.FormatTime)
		if ok := pipeline.Query(pos); !ok {
			//fmt.Println("Failed to query position from pipeline")
			continue
		}

		dur := gst.NewDurationQuery(gst.FormatTime)
		if ok := pipeline.Query(dur); !ok {
			fmt.Println("Failed to query duration from pipeline")
		}

		_, posVal := pos.ParsePosition() //  If either of the above queries failed, these values
		_, durVal := dur.ParseDuration() //  will be 0.
		posDur := time.Duration(posVal) * time.Nanosecond
		durDur := time.Duration(durVal) * time.Nanosecond

		// currentTimeCode.setCode(posDur)

		fmt.Println(posDur, "/", durDur)
	}
}
