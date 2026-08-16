module github.com/LingByte/ling-base/synthesizer/local

go 1.26.2

require (
	github.com/LingByte/ling-base/synthesizer v0.0.0
	github.com/sirupsen/logrus v1.9.3
)

require golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect

replace github.com/LingByte/ling-base/synthesizer => ../
