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
		compositor background=black operator=2   async-handling=true  sync=true latency=5000000000  name=videomix !  timeoverlay !  clockoverlay halignment=right valignment=center text="Scheduler" shaded-background=true font-desc="Sans, 12" !  tee name=vo  
		audiomixer  async-handling=true sync=true  latency=5000000000   name=audiomix async=true !    tee name=au 

	

		hlssink2  async-handling=true name=output location=out/hlssink.%05d.ts playlist-location=out/out.m3u8  sync=true target-duration=10 playlist-length=5 max-files=5 name=hls 
		vo. 
				! queue ! videoconvert ! videoscale ! videorate
				!   x264enc cabac=false  key-int-max=50 quantizer=0  pass=0 speed-preset=ultrafast  tune=zerolatency bitrate=8000   byte-stream=true ! video/x-h264,width=1920,height=1080,pixel-aspect-ratio=1/1,framerate=25/1,format=RGBA,profile=constrained-baseline ! hls.video
		au. 
				! queue ! audioconvert
				! audiorate
				! audioresample
				! audio/x-raw, rate=44100, layout=interleaved, format=F32LE, channels=2 ! avenc_aac bitrate=128000 ac=2
				! audio/mpeg
                ! hls.audio


		audiotestsrc is-live=true wave=1  volume=0.0 !  audioconvert ! audioresample !  audio/x-raw,format=S32LE,rate=44100,channels=2 ! audiomix.
		videotestsrc is-live=true ! videoconvert !  video/x-raw,width=1920,height=1080,pixel-aspect-ratio=1/1,framerate=25/1,format=RGBA,profile=constrained-baseline ! videomix.
		
				`
	gst.Init(nil)
	//filesrc  name=bgloop location="1.mp4" ! decodebin ! videoconvert ! videoscale ! videorate ! video/x-raw,width=1920,height=1080,pixel-aspect-ratio=1/1,framerate=25/1,format=RGBA,profile=constrained-baseline ! videomix.
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

	// //m, _ := pipeline.GetElementByName("bgloop")
	pipeline.GetBus().AddWatch(func(msg *gst.Message) bool {
		if msg.Source() == "mybin1" || msg.Source() == "dynamic1" {
			fmt.Println(msg)
		}
		return true
	})

	go func() {
		time.Sleep(time.Second * 5)
		AddNewUridecodeBin(pipeline)
		// time.Sleep(time.Second * 10)

		// mixer, _ := pipeline.GetElementByName("dynamic1")
		// mixer.GetBus().AddSignalWatch()
		// mixer.SendEvent(gst.NewEOSEvent())
		// // pads, _ := mixer.GetSrcPads()
		// for _, pa := range pads {
		// 	paa := mixer.RemovePad(pa)
		// 	fmt.Printf("%#v \n", paa)

		// }

		// fmt.Println("REMOVED")
		// pipeline.DebugBinToDotFile(gst.DebugGraphShowAll, "AFTERREMOVE")
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
	//var sinkPad *gst.Pad
	bin := gst.NewBin("mybin1")
	var blockPadVideo *gst.Pad
	var blockPadAudio *gst.Pad

	els, _ := gst.NewElementMany("uridecodebin", "videoconvert", "videoscale", "audioconvert", "audiorate", "audioresample", "volume")

	uridecodebin := els[0]
	uridecodebin.Set("name", "dynamic1")
	//uridecodebin.Set("uri", "file:///Users/hjanuschka/pla/1.mp4")
	uridecodebin.Set("uri", "https://mediaviod-bitmovin.krone.at/nogeo/videos/2021/03/19/6adb78babd2fe95-210319_Adabei_Prime_24min_web/stream.m3u8")
	uridecodebin.Set("sync", true)
	uridecodebin.Set("async", true)
	uridecodebin.Set("async-handling", true)
	uridecodebin.SetState(gst.StatePaused)

	bin.Add(uridecodebin)

	pipeline.Add(bin.Element)
	glib.TimeoutAdd(40000, func() {
		fmt.Println("Going to Remove Element")
		s := bin.GetState()
		fmt.Println(s)
		pipeline.Remove(bin.Element)

		vm, _ := pipeline.GetElementByName("videomix")
		pads, _ := vm.GetSinkPads()
		for x, v := range pads {
			fmt.Printf("XPAD: %d\n", x)
			fmt.Printf("%#v\n", v)
			//fmt.Printf("%#v\n", v.IsLinked())
			if v.IsLinked() == false {
				vm.RemovePad(v)
			}
		}
		vm, _ = pipeline.GetElementByName("audiomix")
		pads, _ = vm.GetSinkPads()
		for x, v := range pads {
			fmt.Printf("XPAD: %d\n", x)
			fmt.Printf("%#v\n", v)
			//fmt.Printf("%#v\n", v.IsLinked())
			if v.IsLinked() == false {
				vm.RemovePad(v)
			}
		}
		pipeline.DebugBinToDotFile(gst.DebugGraphShowAll, "AFTER_UNLINK")

		//pipeline.DebugBinToDotFile(gst.DebugGraphShowAll, "AFTER_UNLINK")
		// blockPadVideo.AddProbe(gst.PadProbeTypeBlockDownstream, func(self *gst.Pad, info *gst.PadProbeInfo) gst.PadProbeReturn {
		// 	fmt.Println("VIDEO PAD IS BLOCKED NOW")
		// 	pipeline.Remove(bin.Element)

		// 	vm, _ := pipeline.GetElementByName("videomix")
		// 	pads, _ := vm.GetSinkPads()
		// 	for x, v := range pads {
		// 		fmt.Printf("XPAD: %d\n", x)
		// 		fmt.Printf("%#v\n", v)
		// 		//fmt.Printf("%#v\n", v.IsLinked())
		// 		if v.IsLinked() == false {
		// 			vm.RemovePad(v)
		// 		}
		// 	}
		// 	vm, _ = pipeline.GetElementByName("audiomix")
		// 	pads, _ = vm.GetSinkPads()
		// 	for x, v := range pads {
		// 		fmt.Printf("XPAD: %d\n", x)
		// 		fmt.Printf("%#v\n", v)
		// 		//fmt.Printf("%#v\n", v.IsLinked())
		// 		if v.IsLinked() == false {
		// 			vm.RemovePad(v)
		// 		}
		// 	}
		// 	pipeline.DebugBinToDotFile(gst.DebugGraphShowAll, "AFTER_UNLINK")
		// 	return gst.PadProbeOK
		// })
	})

	bin.SetState(gst.StatePaused)
	//alreadyRun := false
	uridecodebin.Connect("pad-added", func(_ *gst.Element, srcPad *gst.Pad) {
		cur := time.Now().Local()
		future := time.Now().Add(time.Second * 10)

		diff := future.Sub(cur)
		fmt.Println(diff)

		fmt.Println("...PAD INCOMMING")
		srcPad.SetOffset(diff.Nanoseconds())
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
			bin.AddMany(video...)
			for _, e := range video {
				e.SyncStateWithParent()
			}
			videoconvert := video[0]
			videoscale := video[1]

			videoconvertSink := videoconvert.GetStaticPad("sink")
			srcPad.Link(videoconvertSink)
			blockPadVideo = srcPad
			// sinkPad = srcPad
			uridecodebin.Link(videoconvert)
			videoconvert.Link(videoscale)

			//video/x-h264, stream-format=(string)avc, pixel-aspect-ratio=(fraction)1/1, width=(int)1280, height=(int)720, framerate=(fraction)25/1, chroma-format=(string)4:2:0, bit-depth-luma=(uint)8, bit-depth-chroma=(uint)8, parsed=(boolean)true, alignment=(string)au, profile=(string)constrained-baseline, level=(string)3.1, codec_data=(buffer)0142c01fffe100196742c01fd9805005bb011000000300100000030328f183268001000568c9632c80 in anything we support

			videomixer, err := pipeline.GetElementByName("videomix")
			if err != nil {
				panic(err)
			}
			videoscale.LinkFiltered(videomixer, gst.NewCapsFromString("video/x-raw,width=1920,height=1080"))

			fmt.Println("Video Added")
			pipeline.DebugBinToDotFile(gst.DebugGraphShowAll, "VIDEOWORKS")
			srcPad.AddProbe(gst.PadProbeTypeEventDownstream, func(p *gst.Pad, ppi *gst.PadProbeInfo) gst.PadProbeReturn {
				evnt := ppi.GetEvent()
				fmt.Println(evnt.Type() == gst.EventTypeEOS)

				pos := gst.NewPositionQuery(gst.FormatTime)
				if ok := bin.Query(pos); !ok {
					//fmt.Println("Failed to query position from pipeline")
					return gst.PadProbePass
				}

				dur := gst.NewDurationQuery(gst.FormatTime)
				if ok := bin.Query(dur); !ok {
					fmt.Println("Failed to query duration from pipeline")
					return gst.PadProbePass
				}

				_, posVal := pos.ParsePosition() //  If either of the above queries failed, these values
				_, durVal := dur.ParseDuration() //  will be 0.
				posDur := time.Duration(posVal) * time.Nanosecond
				durDur := time.Duration(durVal) * time.Nanosecond

				// currentTimeCode.setCode(posDur)

				fmt.Println("BIN POS:", posDur, "/", durDur)

				return gst.PadProbePass
			})

		}
		if isAudio {

			audio, _ := gst.NewElementMany("audioconvert", "audiorate", "audioresample", "volume")
			bin.AddMany(audio...)
			for _, e := range audio {
				e.SyncStateWithParent()
			}
			audioconvert := audio[0]
			audiorate := audio[1]
			audioresample := audio[2]
			volume := audio[3]
			volume.Set("name", "dvolume1")
			audioconvertSink := audioconvert.GetStaticPad("sink")
			srcPad.Link(audioconvertSink)
			blockPadAudio = srcPad

			uridecodebin.Link(audioconvert)

			e1 := audioconvert.Link(audiorate)
			fmt.Println(e1)
			e1 = audiorate.Link(audioresample)
			fmt.Println(e1)
			e1 = audioresample.Link(volume)
			fmt.Println(e1)

			audiomixer, err := pipeline.GetElementByName("audiomix")
			if err != nil {
				panic(err)
			}
			e2 := volume.Link(audiomixer)
			fmt.Println(e2)
			fmt.Println("Audio Added")

			//uridecodebin.SetState(gst.StatePlaying)
			// for _, e := range audio {
			// 	e.SyncStateWithParent()
			// }
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
