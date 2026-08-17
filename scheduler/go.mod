module github.com/LingByte/ling-base/scheduler

go 1.26.2

require (
	github.com/LingByte/ling-base/common/cron v0.0.0-00010101000000-000000000000
	github.com/LingByte/ling-base/common/lock v0.0.0-00010101000000-000000000000
)

replace (
	github.com/LingByte/ling-base/common/cron => ../common/cron
	github.com/LingByte/ling-base/common/lock => ../common/lock
)
