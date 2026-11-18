# devices

Local audio capture/playback helpers built on [malgo](https://github.com/gen2brain/malgo).

## Features

- List capture and playback devices
- Record and play PCM via `AudioDevice`
- Streaming capture/playback contexts and `StreamAudioPlayer`

## Quick start

```go
import "github.com/LingByte/ling-base/common/devices"

dev, err := devices.NewAudioDevice(devices.DefaultDeviceConfig())
if err != nil {
    return err
}
defer dev.Close()
```
