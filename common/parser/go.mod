module github.com/LingByte/ling-base/common/parser

go 1.26.2

require (
	github.com/BurntSushi/toml v1.6.0
	github.com/LingByte/ling-base/ocr v0.0.0
	github.com/LingByte/ling-base/recognizer v0.0.0
	github.com/hajimehoshi/go-mp3 v0.3.4
	github.com/jung-kurt/gofpdf v1.16.2
	github.com/ledongthuc/pdf v0.0.0-20250511090121-5959a4027728
	github.com/richardlehane/mscfb v1.0.4
	github.com/stretchr/testify v1.11.1
	github.com/xuri/excelize/v2 v2.9.0
	golang.org/x/image v0.45.0
	golang.org/x/net v0.57.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/richardlehane/msoleps v1.0.4 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/xuri/efp v0.0.0-20240408161823-9ad904a10d6d // indirect
	github.com/xuri/nfp v0.0.0-20240318013403-ab9948c2c4a7 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace github.com/LingByte/ling-base => ../../

replace github.com/LingByte/ling-base/ocr => ../../ocr

replace github.com/LingByte/ling-base/recognizer => ../../recognizer
