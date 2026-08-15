module github.com/LingByte/ling-base/mq/factory

go 1.26.2

require (
	github.com/LingByte/ling-base/mq v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/mq/activemq v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/mq/kafka v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/mq/rabbitmq v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/mq/redisstream v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/mq/rocketmq v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/apache/rocketmq-client-go/v2 v2.1.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/go-stomp/stomp/v3 v3.1.5 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rabbitmq/amqp091-go v1.12.0 // indirect
	github.com/redis/go-redis/v9 v9.22.0 // indirect
	github.com/segmentio/kafka-go v0.4.50 // indirect
	github.com/sirupsen/logrus v1.4.0 // indirect
	github.com/tidwall/gjson v1.13.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9 // indirect
	golang.org/x/sys v0.30.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	stathat.com/c/consistent v1.0.0 // indirect
)

replace (
	github.com/LingByte/ling-base/mq => ../
	github.com/LingByte/ling-base/mq/activemq => ../activemq
	github.com/LingByte/ling-base/mq/kafka => ../kafka
	github.com/LingByte/ling-base/mq/rabbitmq => ../rabbitmq
	github.com/LingByte/ling-base/mq/redisstream => ../redisstream
	github.com/LingByte/ling-base/mq/rocketmq => ../rocketmq
)
