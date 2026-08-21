# audioutil

Audio processing utilities for WAV and MP3 files, providing decode, encode, volume, trim, mix, resample, and effects.

## Features

- WAV read/write: 8/16/24/32-bit PCM, mono/stereo
- MP3 decode (via go-mp3); encoding not supported
- Volume: gain, fade in/out, normalize
- Trim: silence removal, range cut
- Mix: merge multiple audio streams
- Resample: linear interpolation
- Convert: mono/stereo, bit depth, WAV/MP3 round-trip
- Effects: fade, speed change, reverse

## Key types

- `Audio` -- in-memory audio buffer (`SampleRate`, `Channels`, `Format`, `Samples`)
- `SampleFormat` -- bit depth constant (`Format8Bit`, `Format16Bit`, `Format24Bit`, `Format32Bit`)
- `Info` -- audio file metadata

## Key functions

- `Load(path)` / `LoadWAV(path)` / `LoadMP3(path)` -- decode audio files
- `SaveWAV(audio, path)` / `Save(audio, path)` -- encode to WAV
- `AdjustVolume`, `TrimSilence`, `Normalize`, `Mix`, `Resample`, `FadeIn`, `FadeOut`
- `ToMono`, `ToStereo`, `ChangeBitDepth`
- `Reverse`, `ChangeSpeed`

## Quick start

```go
import "github.com/LingByte/ling-base/common/audioutil"

// Load WAV or MP3 (auto-detected by extension)
audio, err := audioutil.Load("input.mp3")
if err != nil {
    log.Fatal(err)
}
audio = audioutil.AdjustVolume(audio, 2.0)
audio = audioutil.TrimSilence(audio, -40)
_ = audioutil.SaveWAV(audio, "output.wav")
```

## License

MIT
