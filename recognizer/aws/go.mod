module github.com/LingByte/ling-base/recognizer/aws

go 1.26.2

require (
	github.com/LingByte/ling-base/recognizer v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.30.3
	github.com/aws/aws-sdk-go-v2/config v1.27.27
	github.com/aws/aws-sdk-go-v2/credentials v1.17.27
	github.com/aws/aws-sdk-go-v2/service/transcribestreaming v1.38.3
	github.com/sirupsen/logrus v1.9.3
)

replace github.com/LingByte/ling-base/recognizer => ../
