module github.com/LingByte/ling-base/common/payment/stripe

go 1.26.2

require (
	github.com/LingByte/ling-base/common/payment v0.0.0
	github.com/stretchr/testify v1.7.0
	github.com/stripe/stripe-go/v81 v81.4.0
)

require (
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

replace github.com/LingByte/ling-base/common/payment => ../
