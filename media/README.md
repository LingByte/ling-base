# media

Realtime media pipeline for telephony / WebRTC (sessions, codecs, denoise, AEC).

Module: `github.com/LingByte/ling-base/media`

| Package | Path | Role |
|---------|------|------|
| `media` | `.` | Session, router, packets, resampling |
| `encoder` | `./encoder` | PCM / G.711 / G.722 / Opus (+ optional Rust `lecodec`) |
| `denoise` | `./denoise` | Simple / LeDenoise / SNR helpers |
| `rnnoise` | `./rnnoise` | RNNoise CGO wrapper |
| `aec` | `./aec` | NLMS acoustic echo cancellation |

```bash
go get github.com/LingByte/ling-base/media
```

Optional Rust codec library:

```bash
./encoder/lecodec/build.sh
go test -tags lingcodec ./...
```

> Note: file/image/video utilities live under `common/media/{audio,image,video}` — different modules.
