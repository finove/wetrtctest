package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/ivfreader"
	"github.com/pion/webrtc/v3/pkg/media/oggreader"
)

// Allows compressing offer/answer to bypass terminal input limits.
const compress = false

// MustReadStdin blocks until input is received from stdin
func MustReadStdin() string {
	r := bufio.NewReader(os.Stdin)

	var in string
	for {
		var err error
		in, err = r.ReadString('\n')
		if err != io.EOF {
			if err != nil {
				panic(err)
			}
		}
		in = strings.TrimSpace(in)
		if len(in) > 0 {
			break
		}
	}

	fmt.Println("")

	return in
}

// Encode encodes the input in base64
// It can optionally zip the input before encoding
func Encode(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	if compress {
		b = zip(b)
	}

	return base64.StdEncoding.EncodeToString(b)
}

// Decode decodes the input from base64
// It can optionally unzip the input after decoding
func Decode(in string, obj interface{}) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}

	if compress {
		b = unzip(b)
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		panic(err)
	}
}

func zip(in []byte) []byte {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write(in)
	if err != nil {
		panic(err)
	}
	err = gz.Flush()
	if err != nil {
		panic(err)
	}
	err = gz.Close()
	if err != nil {
		panic(err)
	}
	return b.Bytes()
}

func unzip(in []byte) []byte {
	var b bytes.Buffer
	_, err := b.Write(in)
	if err != nil {
		panic(err)
	}
	r, err := gzip.NewReader(&b)
	if err != nil {
		panic(err)
	}
	res, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return res
}

func SendOggAudio(ctx context.Context, fileName string, audioTrack *webrtc.TrackLocalStaticSample) (err error) {
	var file *os.File
	var ogg *oggreader.OggReader
	var lastGranule uint64
	var oggPageDuration = time.Millisecond * 20
	if file, err = os.Open(fileName); err != nil {
		return
	}
	if ogg, _, err = oggreader.NewWith(file); err != nil {
		return
	}
	ticker := time.NewTicker(oggPageDuration)
OUT:
	for {
		select {
		case <-ticker.C:
			var pageData []byte
			var pageHeader *oggreader.OggPageHeader
			pageData, pageHeader, err = ogg.ParseNextPage()
			if err == io.EOF {
				file.Seek(0, io.SeekStart)
				ogg, _, err = oggreader.NewWith(file)
				if err != nil {
					break OUT
				} else {
					continue
				}
			}
			if err != nil {
				break OUT
			}
			sampleCount := float64(pageHeader.GranulePosition - lastGranule)
			lastGranule = pageHeader.GranulePosition
			simpleDuration := time.Duration((sampleCount/48000)*1000) * time.Millisecond
			if err = audioTrack.WriteSample(media.Sample{Data: pageData, Duration: simpleDuration}); err != nil {
				break OUT
			}
		case <-ctx.Done():
			break OUT
		}
	}
	log.Printf("audio done %v", err)
	return
}

func SendVP8Video(ctx context.Context, fileName string, videoTrack *webrtc.TrackLocalStaticSample) (err error) {
	var file *os.File
	var header *ivfreader.IVFFileHeader
	var ivf *ivfreader.IVFReader
	if file, err = os.Open(fileName); err != nil {
		return
	}
	if ivf, header, err = ivfreader.NewWith(file); err != nil {
		return
	}
	ticker := time.NewTicker(time.Millisecond * time.Duration((float32(header.TimebaseNumerator)/float32(header.TimebaseDenominator))*1000))
OUT:
	for {
		select {
		case <-ticker.C:
			var frame []byte
			frame, _, err = ivf.ParseNextFrame()
			if err == io.EOF {
				file.Seek(0, io.SeekStart)
				ivf, _, err = ivfreader.NewWith(file)
				if err != nil {
					break OUT
				} else {
					continue
				}
			}
			if err != nil {
				break OUT
			}
			if err = videoTrack.WriteSample(media.Sample{Data: frame, Duration: time.Second}); err != nil {
				break OUT
			}
		case <-ctx.Done():
			break OUT
		}
	}
	log.Printf("video done %v", err)
	return
}
