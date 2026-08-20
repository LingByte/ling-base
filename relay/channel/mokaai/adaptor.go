package mokaai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/LingByte/ling-base/relay/channel"
	common "github.com/LingByte/ling-base/relay/common"
	relaymode "github.com/LingByte/ling-base/relay/relaymode"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
	"github.com/LingByte/ling-base/relay/relaykit/types"

)

type Adaptor struct {
}

func (a *Adaptor) ConvertGeminiRequest(context.Context, *common.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(context.Context, *common.RelayInfo, *dto.ClaudeRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertAudioRequest(c context.Context, info *common.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertImageRequest(c context.Context, info *common.RelayInfo, request dto.ImageRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertEmbeddingRequest(c context.Context, info *common.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	//TODO implement me
	return request, nil
}

func (a *Adaptor) Init(info *common.RelayInfo) {

}

func (a *Adaptor) GetRequestURL(info *common.RelayInfo) (string, error) {
	// https://cloud.baidu.com/doc/WENXINWORKSHOP/s/clntwmv7t
	suffix := "chat/"
	if strings.HasPrefix(info.UpstreamModelName, "m3e") {
		suffix = "embeddings"
	}
	fullRequestURL := fmt.Sprintf("%s/%s", info.ChannelBaseUrl, suffix)
	return fullRequestURL, nil
}

func (a *Adaptor) SetupRequestHeader(c context.Context, req *http.Header, info *common.RelayInfo) error {
	channel.SetupApiRequestHeader(info, req)
	req.Set("Authorization", fmt.Sprintf("Bearer %s", info.ApiKey))
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c context.Context, info *common.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	switch info.RelayMode {
	case relaymode.RelayModeEmbeddings:
		baiduEmbeddingRequest := embeddingRequestOpenAI2Moka(*request)
		return baiduEmbeddingRequest, nil
	default:
		return nil, errors.New("not implemented")
	}
}

func (a *Adaptor) ConvertRerankRequest(c context.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c context.Context, info *common.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	// TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) DoRequest(c context.Context, info *common.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoApiRequest(c, a, info, requestBody)
}

func (a *Adaptor) DoResponse(c context.Context, resp *http.Response, info *common.RelayInfo, w http.ResponseWriter) (usage any, err *types.NewAPIError) {

	switch info.RelayMode {
	case relaymode.RelayModeEmbeddings:
		return mokaEmbeddingHandler(c, info, resp, w)
	default:
		// TODO: not supported in library mode
		// err, usage = mokaHandler(c, resp)

	}
	return
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
