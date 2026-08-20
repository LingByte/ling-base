package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/LingByte/ling-base/relay/channel"
	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/relaymode"
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
	panic("implement me")
}

func (a *Adaptor) Init(info *common.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *common.RelayInfo) (string, error) {
	switch info.RelayMode {
	case constant.RelayModeChatCompletions:
		return fmt.Sprintf("%s/client/v4/accounts/%s/ai/v1/chat/completions", info.ChannelBaseUrl, info.ApiVersion), nil
	case constant.RelayModeEmbeddings:
		return fmt.Sprintf("%s/client/v4/accounts/%s/ai/v1/embeddings", info.ChannelBaseUrl, info.ApiVersion), nil
	case constant.RelayModeResponses:
		return fmt.Sprintf("%s/client/v4/accounts/%s/ai/v1/responses", info.ChannelBaseUrl, info.ApiVersion), nil
	default:
		return fmt.Sprintf("%s/client/v4/accounts/%s/ai/run/%s", info.ChannelBaseUrl, info.ApiVersion, info.UpstreamModelName), nil
	}
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
	case constant.RelayModeCompletions:
		return convertCf2CompletionsRequest(*request), nil
	default:
		return request, nil
	}
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c context.Context, info *common.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) DoRequest(c context.Context, info *common.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoApiRequest(c, a, info, requestBody)
}

func (a *Adaptor) ConvertRerankRequest(c context.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c context.Context, info *common.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	return request, nil
}

func (a *Adaptor) ConvertAudioRequest(c context.Context, info *common.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	// TODO: not supported in library mode
	return nil, errors.New("audio request not supported in library mode")
}

func (a *Adaptor) ConvertImageRequest(c context.Context, info *common.RelayInfo, request dto.ImageRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) DoResponse(c context.Context, resp *http.Response, info *common.RelayInfo, w http.ResponseWriter) (usage any, err *types.NewAPIError) {
	switch info.RelayMode {
	case constant.RelayModeEmbeddings:
		fallthrough
	case constant.RelayModeChatCompletions:
		if info.IsStream {
			err, usage = cfStreamHandler(c, info, resp, w)
		} else {
			err, usage = cfHandler(c, info, resp, w)
		}
	case constant.RelayModeResponses:
		// TODO: not supported in library mode
		// if info.IsStream {
		// 	usage, err = openai.OaiResponsesStreamHandler(c, info, resp)
		// } else {
		// 	usage, err = openai.OaiResponsesHandler(c, info, resp)
		// }
		return nil, types.NewOpenAIError(fmt.Errorf("responses mode not supported in library mode"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	case constant.RelayModeAudioTranslation:
		fallthrough
	case constant.RelayModeAudioTranscription:
		err, usage = cfSTTHandler(c, info, resp, w)
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
