module github.com/LingByte/ling-base/voice/recognizer/voiceapi

go 1.26.2

require (
	github.com/LingByte/ling-base/voice/recognizer v0.1.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/sirupsen/logrus v1.9.3
)

require (
	golang.org/x/sys v0.47.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/LingByte/ling-base/voice/recognizer => ../
