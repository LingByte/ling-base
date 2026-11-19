//go:build !lingcodec

package encoder

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/ling-base/media"
)

func createPCMADecode(src, pcm media.CodecConfig) media.EncoderFunc {
	sourceSampleRate := src.SampleRate
	if sourceSampleRate == 0 {
		sourceSampleRate = 8000
	}
	res := media.DefaultResampler(sourceSampleRate, pcm.SampleRate)
	// Reused across frames: Write() copies/converts into the resampler, so
	// this scratch must not escape into the returned MediaPacket.
	var pcmScratch []byte
	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}
		need := len(audioPacket.Payload) << 1
		if cap(pcmScratch) < need {
			pcmScratch = make([]byte, need)
		} else {
			pcmScratch = pcmScratch[:need]
		}
		pcma2pcmInto(pcmScratch, audioPacket.Payload)
		if _, err := res.Write(pcmScratch); err != nil {
			return nil, err
		}
		data := res.Samples()
		if data == nil {
			return nil, nil
		}
		audioPacket.Payload = data
		return []media.MediaPacket{audioPacket}, nil
	}
}

func createPCMAEncode(src, pcm media.CodecConfig) media.EncoderFunc {
	// Use configured target sample rate, if not set use PCMA standard sample rate 8000Hz
	targetSampleRate := src.SampleRate
	if targetSampleRate == 0 {
		targetSampleRate = 8000 // PCMA standard sample rate
	}
	res := media.DefaultResampler(pcm.SampleRate, targetSampleRate)

	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}
		if _, err := res.Write(audioPacket.Payload); err != nil {
			return nil, err
		}
		data := res.Samples()
		if data == nil {
			return nil, nil
		}
		data, err := Pcm2pcma(data)
		if err != nil {
			return nil, err
		}
		return splitFrames(data, &src), nil
	}
}
