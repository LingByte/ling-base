module github.com/LingByte/ling-base/stores/cos

go 1.26.2

require (
	github.com/LingByte/ling-base/stores v0.0.0
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdn v1.3.154
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.3.164
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/monitor v1.3.164
	github.com/tencentyun/cos-go-sdk-v5 v0.7.75
)

require (
	github.com/clbanning/mxj v1.8.4 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mozillazg/go-httpheader v0.2.1 // indirect
)

replace github.com/LingByte/ling-base/stores => ../
