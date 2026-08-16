module github.com/LingByte/ling-base/recognizer/qcloud

go 1.26.2

require (
	github.com/LingByte/ling-base/recognizer v0.0.0
	github.com/matoous/go-nanoid v1.5.0
	github.com/sirupsen/logrus v1.9.3
	github.com/tencentcloud/tencentcloud-speech-sdk-go v1.0.25
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/LingByte/ling-base/recognizer => ../
