module github.com/LingByte/ling-base/common/payment/stripe

go 1.26.2

require (
	github.com/LingByte/ling-base/common/payment v0.0.0
	github.com/stretchr/testify v1.12.1
	github.com/stripe/stripe-go/v81 v81.4.0
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

replace github.com/LingByte/ling-base/common/payment => ../
