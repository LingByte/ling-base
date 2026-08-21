module github.com/LingByte/ling-base/voice/recognizer/gladia

go 1.26.2

require (
	github.com/LingByte/ling-base/voice/recognizer v0.1.1
	github.com/gorilla/websocket v1.5.3
	github.com/sirupsen/logrus v1.9.3
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/stretchr/testify v1.12.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/LingByte/ling-base/voice/recognizer => ../
