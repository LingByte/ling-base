module github.com/LingByte/ling-base/synthesizer/volcengine

go 1.26.2

require (
	github.com/LingByte/ling-base/synthesizer v0.0.0
	github.com/carlmjohnson/requests v0.24.2
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/sirupsen/logrus v1.9.3
)

require (
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/LingByte/ling-base/synthesizer => ../
