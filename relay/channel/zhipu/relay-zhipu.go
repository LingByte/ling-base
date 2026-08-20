package zhipu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/ling-base/relay/constant"
	common "github.com/LingByte/ling-base/relay/common"
	helper2 "github.com/LingByte/ling-base/relay/helper"
	"github.com/LingByte/ling-base/relay/relaykit/dto"
	"github.com/LingByte/ling-base/relay/relaykit/types"
	"github.com/samber/lo"

	"github.com/golang-jwt/jwt/v5"
)

// https://open.bigmodel.cn/doc/api#chatglm_std
// chatglm_std, chatglm_lite
// https://open.bigmodel.cn/api/paas/v3/model-api/chatglm_std/invoke
// https://open.bigmodel.cn/api/paas/v3/model-api/chatglm_std/sse-invoke

var zhipuTokens sync.Map
var expSeconds int64 = 24 * 3600

func getZhipuToken(apikey string) string {
	data, ok := zhipuTokens.Load(apikey)
	if ok {
		tokenData := data.(zhipuTokenData)
		if time.Now().Before(tokenData.ExpiryTime) {
			return tokenData.Token
		}
	}

	split := strings.Split(apikey, ".")
	if len(split) != 2 {
		fmt.Println("invalid zhipu key: " + apikey)
		return ""
	}

	id := split[0]
	secret := split[1]

	expMillis := time.Now().Add(time.Duration(expSeconds)*time.Second).UnixNano() / 1e6
	expiryTime := time.Now().Add(time.Duration(expSeconds) * time.Second)

	timestamp := time.Now().UnixNano() / 1e6

	payload := jwt.MapClaims{
		"api_key":   id,
		"exp":       expMillis,
		"timestamp": timestamp,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	token.Header["alg"] = "HS256"
	token.Header["sign_type"] = "SIGN"

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return ""
	}

	zhipuTokens.Store(apikey, zhipuTokenData{
		Token:      tokenString,
		ExpiryTime: expiryTime,
	})

	return tokenString
}

func requestOpenAI2Zhipu(request dto.GeneralOpenAIRequest) *ZhipuRequest {
	messages := make([]ZhipuMessage, 0, len(request.Messages))
	for _, message := range request.Messages {
		if message.Role == "system" {
			messages = append(messages, ZhipuMessage{
				Role:    "system",
				Content: message.StringContent(),
			})
			messages = append(messages, ZhipuMessage{
				Role:    "user",
				Content: "Okay",
			})
		} else {
			messages = append(messages, ZhipuMessage{
				Role:    message.Role,
				Content: message.StringContent(),
			})
		}
	}
	return &ZhipuRequest{
		Prompt:      messages,
		Temperature: request.Temperature,
		TopP:        lo.FromPtrOr(request.TopP, 0),
		Incremental: false,
	}
}

func responseZhipu2OpenAI(response *ZhipuResponse) *dto.OpenAITextResponse {
	fullTextResponse := dto.OpenAITextResponse{
		Id:      response.Data.TaskId,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Choices: make([]dto.OpenAITextResponseChoice, 0, len(response.Data.Choices)),
		Usage:   response.Data.Usage,
	}
	for i, choice := range response.Data.Choices {
		openaiChoice := dto.OpenAITextResponseChoice{
			Index: i,
			Message: dto.Message{
				Role:    choice.Role,
				Content: strings.Trim(choice.Content, "\""),
			},
			FinishReason: "",
		}
		if i == len(response.Data.Choices)-1 {
			openaiChoice.FinishReason = "stop"
		}
		fullTextResponse.Choices = append(fullTextResponse.Choices, openaiChoice)
	}
	return &fullTextResponse
}

func streamResponseZhipu2OpenAI(zhipuResponse string) *dto.ChatCompletionsStreamResponse {
	var choice dto.ChatCompletionsStreamResponseChoice
	choice.Delta.SetContentString(zhipuResponse)
	response := dto.ChatCompletionsStreamResponse{
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   "chatglm",
		Choices: []dto.ChatCompletionsStreamResponseChoice{choice},
	}
	return &response
}

func streamMetaResponseZhipu2OpenAI(zhipuResponse *ZhipuStreamMetaResponse) (*dto.ChatCompletionsStreamResponse, *dto.Usage) {
	var choice dto.ChatCompletionsStreamResponseChoice
	choice.Delta.SetContentString("")
	choice.FinishReason = &constant.FinishReasonStop
	response := dto.ChatCompletionsStreamResponse{
		Id:      zhipuResponse.RequestId,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   "chatglm",
		Choices: []dto.ChatCompletionsStreamResponseChoice{choice},
	}
	return &response, &zhipuResponse.Usage
}

func zhipuStreamHandler(c context.Context, info *common.RelayInfo, resp *http.Response, w http.ResponseWriter) (*dto.Usage, *types.NewAPIError) {
	var usage *dto.Usage
	scanner := helper2.NewStreamScanner(resp.Body)
	dataChan := make(chan string)
	metaChan := make(chan string)
	stopChan := make(chan bool)
	go func() {
		for {
			data, err := scanner.Scan()
			if err != nil {
				break
			}
			lines := strings.Split(data, "\n")
			for i, line := range lines {
				if len(line) < 5 {
					continue
				}
				if line[:5] == "data:" {
					dataChan <- line[5:]
					if i != len(lines)-1 {
						dataChan <- "\n"
					}
				} else if line[:5] == "meta:" {
					metaChan <- line[5:]
				}
			}
		}
		stopChan <- true
	}()
	helper2.SetEventStreamHeaders(w)
	done := false
	for !done {
		select {
		case data := <-dataChan:
			response := streamResponseZhipu2OpenAI(data)
			jsonResponse, err := json.Marshal(response)
			if err != nil {
				fmt.Println("error marshalling stream response: " + err.Error())
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", string(jsonResponse))
		case data := <-metaChan:
			var zhipuResponse ZhipuStreamMetaResponse
			err := json.Unmarshal([]byte(data), &zhipuResponse)
			if err != nil {
				fmt.Println("error unmarshalling stream response: " + err.Error())
				continue
			}
			response, zhipuUsage := streamMetaResponseZhipu2OpenAI(&zhipuResponse)
			jsonResponse, err := json.Marshal(response)
			if err != nil {
				fmt.Println("error marshalling stream response: " + err.Error())
				continue
			}
			usage = zhipuUsage
			fmt.Fprintf(w, "data: %s\n\n", string(jsonResponse))
		case <-stopChan:
			fmt.Fprintf(w, "data: [DONE]\n\n")
			done = true
		}
	}
	resp.Body.Close()
	return usage, nil
}

func zhipuHandler(c context.Context, info *common.RelayInfo, resp *http.Response, w http.ResponseWriter) (*dto.Usage, *types.NewAPIError) {
	var zhipuResponse ZhipuResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	resp.Body.Close()
	err = json.Unmarshal(responseBody, &zhipuResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if !zhipuResponse.Success {
		return nil, types.WithOpenAIError(types.OpenAIError{
			Message: zhipuResponse.Msg,
			Code:    zhipuResponse.Code,
		}, resp.StatusCode)
	}
	fullTextResponse := responseZhipu2OpenAI(&zhipuResponse)
	jsonResponse, err := json.Marshal(fullTextResponse)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, err = w.Write(jsonResponse)
	return &fullTextResponse.Usage, nil
}
