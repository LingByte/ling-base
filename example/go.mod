module github.com/LingByte/ling-base/example

go 1.26.2

require (
	github.com/LingByte/ling-base v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/bootstrap v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/cache v0.1.0
	github.com/LingByte/ling-base/cache/redis v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/circuitbreaker v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common v0.3.0
	github.com/LingByte/ling-base/common/config v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common/convert v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common/idgen v0.1.0
	github.com/LingByte/ling-base/common/imageutil v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common/qrcode v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common/barcode v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common/jwtutil v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common/videoutil v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/lock v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/metrics v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/opentelemetry v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/scheduler v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/tracing v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common/response v0.1.0
	github.com/LingByte/ling-base/common/response/gin v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common/validate v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/constants v0.1.0
	github.com/LingByte/ling-base/eventbus v0.1.0
	github.com/LingByte/ling-base/limiter v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/limiter/count v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/limiter/tokenbucket v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/notification v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/notification/email v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/notification/inbox v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/parser v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/pool v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/queue v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/queue/memory v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/retry v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/search v0.1.0
	github.com/LingByte/ling-base/search/bleve v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/version v0.1.0
	github.com/gin-gonic/gin v1.12.0
	github.com/redis/go-redis/v9 v9.22.0
	go.uber.org/zap v1.28.0
	gorm.io/gorm v1.31.2
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/LingByte/ling-base/recognizer v0.0.0 // indirect
	github.com/RoaringBitmap/roaring/v2 v2.4.5 // indirect
	github.com/bits-and-blooms/bitset v1.22.0 // indirect
	github.com/blevesearch/bleve/v2 v2.5.0 // indirect
	github.com/blevesearch/bleve_index_api v1.2.7 // indirect
	github.com/blevesearch/geo v0.1.20 // indirect
	github.com/blevesearch/go-faiss v1.0.25 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/gtreap v0.1.1 // indirect
	github.com/blevesearch/mmap-go v1.0.4 // indirect
	github.com/blevesearch/scorch_segment_api/v2 v2.3.9 // indirect
	github.com/blevesearch/segment v0.9.1 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/upsidedown_store_api v1.0.2 // indirect
	github.com/blevesearch/vellum v1.1.0 // indirect
	github.com/blevesearch/zapx/v11 v11.4.1 // indirect
	github.com/blevesearch/zapx/v12 v12.4.1 // indirect
	github.com/blevesearch/zapx/v13 v13.4.1 // indirect
	github.com/blevesearch/zapx/v14 v14.4.1 // indirect
	github.com/blevesearch/zapx/v15 v15.4.1 // indirect
	github.com/blevesearch/zapx/v16 v16.2.2 // indirect
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic v1.15.0 // indirect
	github.com/bytedance/sonic/loader v0.5.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emersion/go-imap/v2 v2.0.0-20251216103119-7ac47a9cfd9a // indirect
	github.com/emersion/go-message v0.18.2 // indirect
	github.com/emersion/go-sasl v0.0.0-20241020182733-b788ff22d5a6 // indirect
	github.com/gabriel-vasile/mimetype v1.4.12 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/glebarez/sqlite v1.11.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.1 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/golang/geo v0.0.0-20210211234256-740aa86cb551 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hajimehoshi/go-mp3 v0.3.4 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/ledongthuc/pdf v0.0.0-20250511090121-5959a4027728 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.4 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.1 // indirect
	github.com/xuri/efp v0.0.0-20240408161823-9ad904a10d6d // indirect
	github.com/xuri/excelize/v2 v2.9.0 // indirect
	github.com/xuri/nfp v0.0.0-20240318013403-ab9948c2c4a7 // indirect
	go.etcd.io/bbolt v1.4.0 // indirect
	go.mongodb.org/mongo-driver/v2 v2.5.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/arch v0.22.0 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/mysql v1.6.0 // indirect
	gorm.io/driver/postgres v1.6.2 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
)

replace (
	github.com/LingByte/ling-base => ../
	github.com/LingByte/ling-base/bootstrap => ../bootstrap
	github.com/LingByte/ling-base/cache => ../cache
	github.com/LingByte/ling-base/cache/redis => ../cache/redis
	github.com/LingByte/ling-base/circuitbreaker => ../circuitbreaker
	github.com/LingByte/ling-base/common/config => ../common/config
	github.com/LingByte/ling-base/common/convert => ../common/convert
	github.com/LingByte/ling-base/common/idgen => ../common/idgen
	github.com/LingByte/ling-base/common/imageutil => ../common/imageutil
	github.com/LingByte/ling-base/common/qrcode => ../common/qrcode
	github.com/LingByte/ling-base/common/barcode => ../common/barcode
	github.com/LingByte/ling-base/common/jwtutil => ../common/jwtutil
	github.com/LingByte/ling-base/common/videoutil => ../common/videoutil
	github.com/LingByte/ling-base/lock => ../lock
	github.com/LingByte/ling-base/metrics => ../metrics
	github.com/LingByte/ling-base/opentelemetry => ../opentelemetry
	github.com/LingByte/ling-base/scheduler => ../scheduler
	github.com/LingByte/ling-base/tracing => ../tracing
	github.com/LingByte/ling-base/common/response => ../common/response
	github.com/LingByte/ling-base/common/response/gin => ../common/response/gin
	github.com/LingByte/ling-base/common/validate => ../common/validate
	github.com/LingByte/ling-base/eventbus => ../eventbus
	github.com/LingByte/ling-base/limiter => ../limiter
	github.com/LingByte/ling-base/limiter/count => ../limiter/count
	github.com/LingByte/ling-base/limiter/tokenbucket => ../limiter/tokenbucket
	github.com/LingByte/ling-base/notification => ../notification
	github.com/LingByte/ling-base/notification/email => ../notification/email
	github.com/LingByte/ling-base/notification/inbox => ../notification/inbox
	github.com/LingByte/ling-base/parser => ../parser
	github.com/LingByte/ling-base/pool => ../pool
	github.com/LingByte/ling-base/queue => ../queue
	github.com/LingByte/ling-base/queue/memory => ../queue/memory
	github.com/LingByte/ling-base/recognizer => ../recognizer
	github.com/LingByte/ling-base/retry => ../retry
	github.com/LingByte/ling-base/search => ../search
	github.com/LingByte/ling-base/search/bleve => ../search/bleve
	github.com/LingByte/ling-base/version => ../version
)
