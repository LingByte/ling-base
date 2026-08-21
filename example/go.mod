module github.com/LingByte/ling-base/example

go 1.26.2

require (
	github.com/LingByte/ling-base/agent v0.0.0
	github.com/LingByte/ling-base/bootstrap v0.1.4
	github.com/LingByte/ling-base/common v0.3.0
	github.com/LingByte/ling-base/common/barcode v0.1.0
	github.com/LingByte/ling-base/common/cache v0.1.0
	github.com/LingByte/ling-base/common/cache/redis v0.1.0
	github.com/LingByte/ling-base/common/circuitbreaker v0.1.0
	github.com/LingByte/ling-base/common/config v0.1.2
	github.com/LingByte/ling-base/common/constants v0.1.1
	github.com/LingByte/ling-base/common/convert v0.1.0
	github.com/LingByte/ling-base/common/crypto v0.2.0
	github.com/LingByte/ling-base/common/eventbus v0.1.1
	github.com/LingByte/ling-base/common/idgen v0.1.0
	github.com/LingByte/ling-base/common/imageutil v0.1.0
	github.com/LingByte/ling-base/common/jwtutil v0.2.0
	github.com/LingByte/ling-base/common/limiter v0.1.0
	github.com/LingByte/ling-base/common/limiter/count v0.1.0
	github.com/LingByte/ling-base/common/limiter/tokenbucket v0.1.0
	github.com/LingByte/ling-base/common/lock v0.1.0
	github.com/LingByte/ling-base/common/logger v0.1.0
	github.com/LingByte/ling-base/common/metrics v0.1.0
	github.com/LingByte/ling-base/common/migration v0.1.0
	github.com/LingByte/ling-base/common/migration/gormmigrator v0.1.0
	github.com/LingByte/ling-base/common/notification v0.1.1
	github.com/LingByte/ling-base/common/notification/email v0.1.0
	github.com/LingByte/ling-base/common/notification/inbox v0.1.1
	github.com/LingByte/ling-base/common/opentelemetry v0.1.1
	github.com/LingByte/ling-base/common/parser v0.1.1
	github.com/LingByte/ling-base/common/pool v0.1.0
	github.com/LingByte/ling-base/common/qrcode v0.1.0
	github.com/LingByte/ling-base/common/queue v0.1.0
	github.com/LingByte/ling-base/common/queue/memory v0.1.0
	github.com/LingByte/ling-base/common/response v0.1.0
	github.com/LingByte/ling-base/common/response/gin v0.1.1
	github.com/LingByte/ling-base/common/retry v0.1.0
	github.com/LingByte/ling-base/common/scheduler v0.1.1
	github.com/LingByte/ling-base/common/search v0.1.0
	github.com/LingByte/ling-base/common/search/bleve v0.1.0
	github.com/LingByte/ling-base/common/tracing v0.1.0
	github.com/LingByte/ling-base/common/validate v0.1.0
	github.com/LingByte/ling-base/common/videoutil v0.1.0
	github.com/LingByte/ling-base/version v0.1.0
	github.com/gin-gonic/gin v1.12.0
	github.com/redis/go-redis/v9 v9.22.0
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/metric v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
	go.uber.org/zap v1.28.0
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.2
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/JohannesKaufmann/html-to-markdown v1.6.0 // indirect
	github.com/LingByte/ling-base/common/cron v0.1.0 // indirect
	github.com/LingByte/ling-base/providers/ocr v0.0.0 // indirect
	github.com/LingByte/ling-base/relay v0.0.0 // indirect
	github.com/LingByte/ling-base/relay/relaykit v0.0.0 // indirect
	github.com/LingByte/ling-base/voice/recognizer v0.1.1 // indirect
	github.com/PuerkitoBio/goquery v1.9.2 // indirect
	github.com/RoaringBitmap/roaring/v2 v2.4.5 // indirect
	github.com/alecthomas/chroma/v2 v2.20.0 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/anthropics/anthropic-sdk-go v1.45.0 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.24.4 // indirect
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
	github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect
	github.com/boombuler/barcode v1.1.0 // indirect
	github.com/buger/jsonparser v1.1.2 // indirect
	github.com/bytedance/gopkg v0.1.4 // indirect
	github.com/bytedance/sonic v1.15.2 // indirect
	github.com/bytedance/sonic/loader v0.5.1 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/bubbles v1.0.0 // indirect
	github.com/charmbracelet/bubbletea v1.3.10 // indirect
	github.com/charmbracelet/colorprofile v0.4.1 // indirect
	github.com/charmbracelet/glamour v1.0.0 // indirect
	github.com/charmbracelet/lipgloss v1.1.1-0.20250404203927-76690c660834 // indirect
	github.com/charmbracelet/x/ansi v0.11.6 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.15 // indirect
	github.com/charmbracelet/x/exp/slice v0.0.0-20250327172914-2fdc97757edf // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/chromedp/cdproto v0.0.0-20260427013145-5737772c319b // indirect
	github.com/chromedp/chromedp v0.15.1 // indirect
	github.com/chromedp/sysutil v1.1.0 // indirect
	github.com/clipperhouse/displaywidth v0.9.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/cloudwego/base64x v0.1.7 // indirect
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emersion/go-imap/v2 v2.0.0-20251216103119-7ac47a9cfd9a // indirect
	github.com/emersion/go-message v0.18.2 // indirect
	github.com/emersion/go-sasl v0.0.0-20241020182733-b788ff22d5a6 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/gin-contrib/sse v1.1.1 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/glebarez/sqlite v1.11.0 // indirect
	github.com/go-json-experiment/json v0.0.0-20260214004413-d219187c3433 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.3 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/geo v0.0.0-20210211234256-740aa86cb551 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/hajimehoshi/go-mp3 v0.3.4 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/invopop/jsonschema v0.13.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.10.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.13-0.20220915233716-71ac16282d12 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/ledongthuc/pdf v0.0.0-20250511090121-5959a4027728 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/makiuchi-d/gozxing v0.1.1 // indirect
	github.com/mattn/go-isatty v0.0.23 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.24 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/modelcontextprotocol/go-sdk v1.6.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.24.1 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.70.1 // indirect
	github.com/prometheus/otlptranslator v1.0.0 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.60.0 // indirect
	github.com/razvandimescu/gopdf v0.9.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.4 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sahilm/fuzzy v0.1.3 // indirect
	github.com/samber/lo v1.47.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/standard-webhooks/standard-webhooks/libraries v0.0.1 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.1 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/xuri/efp v0.0.0-20240408161823-9ad904a10d6d // indirect
	github.com/xuri/excelize/v2 v2.9.0 // indirect
	github.com/xuri/nfp v0.0.0-20240318013403-ab9948c2c4a7 // indirect
	github.com/yeqown/go-qrcode/v2 v2.3.0 // indirect
	github.com/yeqown/go-qrcode/writer/standard v1.4.0 // indirect
	github.com/yeqown/reedsolomon v1.0.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/yuin/goldmark v1.7.13 // indirect
	github.com/yuin/goldmark-emoji v1.0.6 // indirect
	go.etcd.io/bbolt v1.4.0 // indirect
	go.mongodb.org/mongo-driver/v2 v2.8.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.40.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.40.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.40.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.62.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.16.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.40.0 // indirect
	go.opentelemetry.io/otel/log v0.16.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.16.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/arch v0.29.0 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.45.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260630182238-925bb5da69e7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260807164820-c8921c73eeea // indirect
	google.golang.org/grpc v1.83.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/mysql v1.6.0 // indirect
	gorm.io/driver/postgres v1.6.2 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
	mvdan.cc/sh/v3 v3.13.1 // indirect
)

replace github.com/LingByte/ling-base => ../

replace github.com/LingByte/ling-base/agent => ../agent

replace github.com/LingByte/ling-base/bootstrap => ../bootstrap

replace github.com/LingByte/ling-base/common => ../common

replace github.com/LingByte/ling-base/common/barcode => ../common/barcode

replace github.com/LingByte/ling-base/common/cache => ../common/cache

replace github.com/LingByte/ling-base/common/cache/redis => ../common/cache/redis

replace github.com/LingByte/ling-base/common/circuitbreaker => ../common/circuitbreaker

replace github.com/LingByte/ling-base/common/config => ../common/config

replace github.com/LingByte/ling-base/common/constants => ../common/constants

replace github.com/LingByte/ling-base/common/convert => ../common/convert

replace github.com/LingByte/ling-base/common/eventbus => ../common/eventbus

replace github.com/LingByte/ling-base/common/idgen => ../common/idgen

replace github.com/LingByte/ling-base/common/imageutil => ../common/imageutil

replace github.com/LingByte/ling-base/common/jwtutil => ../common/jwtutil

replace github.com/LingByte/ling-base/common/limiter => ../common/limiter

replace github.com/LingByte/ling-base/common/limiter/count => ../common/limiter/count

replace github.com/LingByte/ling-base/common/limiter/tokenbucket => ../common/limiter/tokenbucket

replace github.com/LingByte/ling-base/common/lock => ../common/lock

replace github.com/LingByte/ling-base/common/metrics => ../common/metrics

replace github.com/LingByte/ling-base/common/migration => ../common/migration

replace github.com/LingByte/ling-base/common/migration/gormmigrator => ../common/migration/gormmigrator

replace github.com/LingByte/ling-base/common/notification => ../common/notification

replace github.com/LingByte/ling-base/common/notification/email => ../common/notification/email

replace github.com/LingByte/ling-base/common/notification/inbox => ../common/notification/inbox

replace github.com/LingByte/ling-base/common/opentelemetry => ../common/opentelemetry

replace github.com/LingByte/ling-base/common/parser => ../common/parser

replace github.com/LingByte/ling-base/common/pool => ../common/pool

replace github.com/LingByte/ling-base/common/qrcode => ../common/qrcode

replace github.com/LingByte/ling-base/common/queue => ../common/queue

replace github.com/LingByte/ling-base/common/queue/memory => ../common/queue/memory

replace github.com/LingByte/ling-base/common/response => ../common/response

replace github.com/LingByte/ling-base/common/response/gin => ../common/response/gin

replace github.com/LingByte/ling-base/common/retry => ../common/retry

replace github.com/LingByte/ling-base/common/scheduler => ../common/scheduler

replace github.com/LingByte/ling-base/common/search => ../common/search

replace github.com/LingByte/ling-base/common/search/bleve => ../common/search/bleve

replace github.com/LingByte/ling-base/common/tracing => ../common/tracing

replace github.com/LingByte/ling-base/common/validate => ../common/validate

replace github.com/LingByte/ling-base/common/videoutil => ../common/videoutil

replace github.com/LingByte/ling-base/providers/ocr => ../providers/ocr

replace github.com/LingByte/ling-base/relay => ../relay

replace github.com/LingByte/ling-base/relay/relaykit => ../relay/relaykit

replace github.com/LingByte/ling-base/version => ../version

replace github.com/LingByte/ling-base/voice/recognizer => ../voice/recognizer
