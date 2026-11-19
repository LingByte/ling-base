//go:build !cgo

package denoise

import "errors"

var errDenoiseNoCGO = errors.New("denoise: unavailable when built with CGO_ENABLED=0")

// DenoiseConfig 降噪配置（无 CGO 构建下的纯 Go 定义）
type DenoiseConfig struct {
	AECEnable     bool
	AGCEnable     bool
	SampleRate    int
	Channels      int
	BitsPerSample int
}

// DenoiseProcessor is a stub when CGO is disabled.
type DenoiseProcessor struct {
	config DenoiseConfig
}

func NewDenoiseProcessor(config *DenoiseConfig) (*DenoiseProcessor, error) {
	return nil, errDenoiseNoCGO
}

func (p *DenoiseProcessor) Process(input []byte) ([]byte, error) {
	return nil, errDenoiseNoCGO
}

func (p *DenoiseProcessor) ProcessInPlace(data []byte) error {
	return errDenoiseNoCGO
}

func (p *DenoiseProcessor) Reset() error {
	return errDenoiseNoCGO
}

func (p *DenoiseProcessor) GetConfig() DenoiseConfig {
	if p == nil {
		return DenoiseConfig{}
	}
	return p.config
}

func (p *DenoiseProcessor) SetAECEnable(enable bool) error {
	return errDenoiseNoCGO
}

func (p *DenoiseProcessor) SetAGCEnable(enable bool) error {
	return errDenoiseNoCGO
}

func (p *DenoiseProcessor) Close() error {
	return nil
}

func Version() string {
	return "unavailable(cgo=0)"
}
