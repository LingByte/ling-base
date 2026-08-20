package suno

import (
	"context"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/LingByte/ling-base/relay/service"
	"github.com/LingByte/ling-base/relay/task/taskcommon"
	common "github.com/LingByte/ling-base/relay/common"
	"github.com/LingByte/ling-base/relay/channel"
)

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
}

// ParseTaskResult is not used for Suno tasks.
// Suno polling uses a dedicated batch-fetch path (service.UpdateSunoTasks) that
// receives dto.TaskResponse[[]dto.SunoDataResponse] from the upstream /fetch API.
// This differs from the per-task polling used by video adaptors.
func (a *TaskAdaptor) ParseTaskResult([]byte) (*common.TaskInfo, error) {
	return nil, fmt.Errorf("suno uses batch polling via UpdateSunoTasks, ParseTaskResult is not applicable")
}

func (a *TaskAdaptor) Init(info *common.RelayInfo) {
	a.ChannelType = info.ChannelType
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c context.Context, info *common.RelayInfo) (taskErr *common.TaskError) {
	// TODO: not supported in library mode
	// action := strings.ToUpper(c.Param("action"))
	//
	// var sunoRequest *dto.SunoSubmitReq
	// err := json.UnmarshalBodyReusable(c, &sunoRequest)
	// if err != nil {
	// 	taskErr = service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	// 	return
	// }
	// err = actionValidate(c, sunoRequest, action)
	// if err != nil {
	// 	taskErr = service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	// 	return
	// }
	//
	// //if sunoRequest.ContinueClipId != "" {
	// //	if sunoRequest.TaskID == "" {
	// //		taskErr = service.TaskErrorWrapperLocal(fmt.Errorf("task id is empty"), "invalid_request", http.StatusBadRequest)
	// //		return
	// //	}
	// //	info.OriginTaskID = sunoRequest.TaskID
	// //}
	//
	// info.Action = action
	// // c.Set("task_request", sunoRequest)
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *common.RelayInfo) (string, error) {
	baseURL := info.ChannelBaseUrl
	fullRequestURL := fmt.Sprintf("%s%s", baseURL, "/suno/submit/"+info.Action)
	return fullRequestURL, nil
}

func (a *TaskAdaptor) BuildRequestHeader(c context.Context, req *http.Request, info *common.RelayInfo) error {
	req.Header.Set("Content-Type", info.RequestHeaders["Content-Type"])
	req.Header.Set("Accept", info.RequestHeaders["Accept"])
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c context.Context, info *common.RelayInfo) (io.Reader, error) {
	// TODO: not supported in library mode
	// sunoRequest, ok := c.Get("task_request")
	// if !ok {
	// 	return nil, fmt.Errorf("task_request not found in context")
	// }
	// data, err := json.Marshal(sunoRequest)
	// if err != nil {
	// 	return nil, err
	// }
	// return bytes.NewReader(data), nil
	return nil, fmt.Errorf("not supported in library mode")
}

func (a *TaskAdaptor) DoRequest(c context.Context, info *common.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c context.Context, resp *http.Response, info *common.RelayInfo) (taskID string, taskData []byte, taskErr *common.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	// TODO: not supported in library mode
	// var sunoResponse dto.TaskResponse[string]
	// err = json.Unmarshal(responseBody, &sunoResponse)
	// if err != nil {
	// 	taskErr = service.TaskErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	// 	return
	// }
	// if !sunoResponse.IsSuccess() {
	// 	taskErr = service.TaskErrorWrapper(fmt.Errorf("%s", sunoResponse.Message), sunoResponse.Code, http.StatusInternalServerError)
	// 	return
	// }
	//
	// // 使用公开 task_xxxx ID 替换上游 ID 返回给客户端
	// publicResponse := dto.TaskResponse[string]{
	// 	Code:    sunoResponse.Code,
	// 	Message: sunoResponse.Message,
	// 	Data:    info.PublicTaskID,
	// }
	// c.JSON(http.StatusOK, publicResponse)

	return string(responseBody), responseBody, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	requestUrl := fmt.Sprintf("%s/suno/fetch", baseUrl)
	byteBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", requestUrl, bytes.NewBuffer(byteBody))
	if err != nil {
		// TODO: not supported in library mode
		// common.SysLog(fmt.Sprintf("Get Task error: %v", err))
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	// TODO: not supported in library mode (proxy ignored)
	return http.DefaultClient.Do(req)
}

func actionValidate(c context.Context, sunoRequest interface{}, action string) (err error) {
	// TODO: not supported in library mode
	// switch action {
	// case constant.SunoActionMusic:
	// 	if sunoRequest.Mv == "" {
	// 		sunoRequest.Mv = "chirp-v3-0"
	// 	}
	// case constant.SunoActionLyrics:
	// 	if sunoRequest.Prompt == "" {
	// 		err = fmt.Errorf("prompt_empty")
	// 		return
	// 	}
	// default:
	// 	err = fmt.Errorf("invalid_action")
	// }
	return
}
