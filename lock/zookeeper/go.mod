module github.com/LingByte/ling-base/lock/zookeeper

go 1.26.2

require (
	github.com/LingByte/ling-base/lock v0.0.0
	github.com/go-zookeeper/zk v1.0.4
)

replace github.com/LingByte/ling-base/lock => ../
