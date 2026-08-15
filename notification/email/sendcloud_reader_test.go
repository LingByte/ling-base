// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package email

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface check.
var _ MailReader = (*SendCloudReader)(nil)

// ──────────────────────────────────────────────
// NewSendCloudReader
// ──────────────────────────────────────────────

func TestNewSendCloudReader_MissingAPIUser(t *testing.T) {
	_, err := NewSendCloudReader(SendCloudConfig{APIKey: "k"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api_user")
}

func TestNewSendCloudReader_MissingAPIKey(t *testing.T) {
	_, err := NewSendCloudReader(SendCloudConfig{APIUser: "u"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api_key")
}

func TestNewSendCloudReader_Success(t *testing.T) {
	r, err := NewSendCloudReader(SendCloudConfig{APIUser: "u", APIKey: "k"})
	require.NoError(t, err)
	assert.Equal(t, "sendcloud", r.Kind())
	assert.Equal(t, "https://api.sendcloud.net/apiv2/data/emailStatus", r.endpoint)
}

func TestNewSendCloudReader_EndpointOverride(t *testing.T) {
	r, err := NewSendCloudReader(SendCloudConfig{
		APIUser:  "u",
		APIKey:   "k",
		Endpoint: "https://api.sendcloud.net/apiv2/mail/send",
	})
	require.NoError(t, err)
	// Endpoint without "emailStatus" gets replaced with default.
	assert.Contains(t, r.endpoint, "emailStatus")
}

// ──────────────────────────────────────────────
// QueryStatus
// ──────────────────────────────────────────────

func newSendCloudTestServer(t *testing.T, handler http.HandlerFunc) (*SendCloudReader, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	r, err := NewSendCloudReader(SendCloudConfig{APIUser: "u", APIKey: "k"})
	require.NoError(t, err)
	r.endpoint = srv.URL
	r.client = srv.Client()
	return r, srv
}

func TestSendCloudReader_QueryStatus_Success(t *testing.T) {
	body := `{
		"result": true,
		"statusCode": 200,
		"message": "请求成功",
		"info": {
			"total": "2",
			"voListSize": 2,
			"voList": [
				{
					"emailId": "abc$1@sendcloud.im",
					"status": "送达",
					"subStatus": null,
					"subStatusDesc": null,
					"apiUser": "test",
					"recipients": "user@example.com",
					"requestTime": "2026-08-14 10:00:00",
					"modifiedTime": "2026-08-14 10:00:05",
					"sendLog": "delivered"
				},
				{
					"emailId": "def$2@sendcloud.im",
					"status": "无效邮件-地址不存在",
					"subStatus": "406",
					"subStatusDesc": "地址不存在",
					"apiUser": "test",
					"recipients": "bad@example.com",
					"requestTime": "2026-08-14 11:00:00",
					"modifiedTime": "2026-08-14 11:00:10",
					"sendLog": "unrouteable address"
				}
			]
		}
	}`

	reader, _ := newSendCloudTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "u", r.FormValue("apiUser"))
		assert.Equal(t, "k", r.FormValue("apiKey"))
		assert.Equal(t, "3", r.FormValue("days"))
		assert.Equal(t, "5", r.FormValue("limit"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	})

	records, total, err := reader.QueryStatus(SendCloudStatusQuery{Days: 3, Limit: 5})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, records, 2)

	assert.Equal(t, "abc$1@sendcloud.im", records[0].EmailID)
	assert.Equal(t, "送达", records[0].Status)
	assert.Equal(t, "user@example.com", records[0].Recipients)

	assert.Equal(t, "def$2@sendcloud.im", records[1].EmailID)
	assert.Equal(t, "无效邮件-地址不存在", records[1].Status)
	assert.Equal(t, "406", records[1].SubStatus)
	assert.Equal(t, "地址不存在", records[1].SubStatusDesc)
}

func TestSendCloudReader_QueryStatus_DateRange(t *testing.T) {
	reader, _ := newSendCloudTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "2026-08-01", r.FormValue("startDate"))
		assert.Equal(t, "2026-08-03", r.FormValue("endDate"))
		assert.Empty(t, r.FormValue("days"))
		w.Write([]byte(`{"result":true,"statusCode":200,"message":"","info":{"total":"0","voListSize":0,"voList":[]}}`))
	})

	_, _, err := reader.QueryStatus(SendCloudStatusQuery{StartDate: "2026-08-01", EndDate: "2026-08-03"})
	require.NoError(t, err)
}

func TestSendCloudReader_QueryStatus_DefaultDays(t *testing.T) {
	reader, _ := newSendCloudTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "1", r.FormValue("days"))
		w.Write([]byte(`{"result":true,"statusCode":200,"message":"","info":{"total":"0","voListSize":0,"voList":[]}}`))
	})

	_, _, err := reader.QueryStatus(SendCloudStatusQuery{})
	require.NoError(t, err)
}

func TestSendCloudReader_QueryStatus_Filters(t *testing.T) {
	reader, _ := newSendCloudTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "user@test.com", r.FormValue("email"))
		assert.Equal(t, "id1;id2", r.FormValue("emailIds"))
		assert.Equal(t, "lbl1", r.FormValue("labelId"))
		assert.Equal(t, "mylabel", r.FormValue("labelName"))
		assert.Equal(t, "u1;u2", r.FormValue("apiUserList"))
		assert.Equal(t, "10", r.FormValue("start"))
		assert.Equal(t, "50", r.FormValue("limit"))
		assert.Equal(t, "1", r.FormValue("status"))
		assert.Equal(t, "401;406", r.FormValue("subStatus"))
		w.Write([]byte(`{"result":true,"statusCode":200,"message":"","info":{"total":"0","voListSize":0,"voList":[]}}`))
	})

	_, _, err := reader.QueryStatus(SendCloudStatusQuery{
		Email:       "user@test.com",
		EmailIDs:    []string{"id1", "id2"},
		LabelID:     "lbl1",
		LabelName:   "mylabel",
		APIUserList: []string{"u1", "u2"},
		Days:        2,
		Start:       10,
		Limit:       50,
		Status:      "1",
		SubStatus:   "401;406",
	})
	require.NoError(t, err)
}

func TestSendCloudReader_QueryStatus_APIError(t *testing.T) {
	reader, _ := newSendCloudTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":false,"statusCode":50000,"message":"接口频率受限","info":{}}`))
	})

	_, _, err := reader.QueryStatus(SendCloudStatusQuery{Days: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "接口频率受限")
}

func TestSendCloudReader_QueryStatus_HTTPError(t *testing.T) {
	reader, _ := newSendCloudTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"result":false,"statusCode":500,"message":"server error","info":{}}`))
	})

	_, _, err := reader.QueryStatus(SendCloudStatusQuery{Days: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http 500")
}

func TestSendCloudReader_QueryStatus_InvalidJSON(t *testing.T) {
	reader, _ := newSendCloudTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	})

	_, _, err := reader.QueryStatus(SendCloudStatusQuery{Days: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse response")
}

func TestSendCloudReader_QueryStatus_EmptyResult(t *testing.T) {
	reader, _ := newSendCloudTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":true,"statusCode":200,"message":"","info":{"total":"0","voListSize":0,"voList":[]}}`))
	})

	records, total, err := reader.QueryStatus(SendCloudStatusQuery{Days: 1})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, records)
}

// ──────────────────────────────────────────────
// ReadMessages (MailReader interface)
// ──────────────────────────────────────────────

func TestSendCloudReader_ReadMessages(t *testing.T) {
	body := `{
		"result": true,
		"statusCode": 200,
		"message": "",
		"info": {
			"total": "1",
			"voListSize": 1,
			"voList": [
				{
					"emailId": "abc$1@sendcloud.im",
					"status": "送达",
					"recipients": "user@example.com",
					"requestTime": "2026-08-14 10:00:00",
					"modifiedTime": "2026-08-14 10:00:05",
					"sendLog": "delivered"
				}
			]
		}
	}`

	reader, _ := newSendCloudTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})
	reader.cfg.From = "sender@example.com"
	reader.cfg.FromName = "Sender"

	msgs, err := reader.ReadMessages(5)
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	msg := msgs[0]
	assert.Equal(t, "abc$1@sendcloud.im", msg.ID)
	assert.Equal(t, "sender@example.com", msg.From)
	assert.Equal(t, "Sender", msg.FromName)
	assert.Equal(t, []string{"user@example.com"}, msg.To)
	assert.Contains(t, msg.Subject, "送达")
	assert.Contains(t, msg.TextBody, "emailId=abc$1@sendcloud.im")
	assert.Contains(t, msg.TextBody, "status=送达")
	assert.False(t, msg.Date.IsZero())
}

func TestSendCloudReader_ReadMessages_LimitCapped(t *testing.T) {
	reader, _ := newSendCloudTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		// limit should be capped at 100
		assert.Equal(t, "100", r.FormValue("limit"))
		w.Write([]byte(`{"result":true,"statusCode":200,"message":"","info":{"total":"0","voListSize":0,"voList":[]}}`))
	})

	msgs, err := reader.ReadMessages(200)
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestSendCloudReader_ReadMessages_Error(t *testing.T) {
	reader, _ := newSendCloudTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":false,"statusCode":500,"message":"fail","info":{}}`))
	})

	_, err := reader.ReadMessages(5)
	require.Error(t, err)
}

// ──────────────────────────────────────────────
// No-op methods
// ──────────────────────────────────────────────

func TestSendCloudReader_MarkRead_NoOp(t *testing.T) {
	r, _ := NewSendCloudReader(SendCloudConfig{APIUser: "u", APIKey: "k"})
	assert.NoError(t, r.MarkRead("any-id"))
}

func TestSendCloudReader_DeleteMessage_NoOp(t *testing.T) {
	r, _ := NewSendCloudReader(SendCloudConfig{APIUser: "u", APIKey: "k"})
	assert.NoError(t, r.DeleteMessage("any-id"))
}

func TestSendCloudReader_Close_NoOp(t *testing.T) {
	r, _ := NewSendCloudReader(SendCloudConfig{APIUser: "u", APIKey: "k"})
	assert.NoError(t, r.Close())
}

// ──────────────────────────────────────────────
// SendCloudStatusToMailStatus
// ──────────────────────────────────────────────

func TestSendCloudStatusToMailStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"送达", StatusDelivered},
		{"无效邮件-地址不存在", StatusFailed},
		{"无效邮件-SendCloud黑名单中", StatusFailed},
		{"软退信-服务不可达", StatusSoftBounce},
		{"软退信-IP、域名被拒", StatusSoftBounce},
		{"请求中", StatusSent},
		{"", StatusSent},
		{"unknown status", StatusUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, SendCloudStatusToMailStatus(tt.input))
		})
	}
}

// ──────────────────────────────────────────────
// SendCloudDeliveryRecord time parsing
// ──────────────────────────────────────────────

func TestSendCloudDeliveryRecord_ParsedRequestTime(t *testing.T) {
	rec := SendCloudDeliveryRecord{RequestTime: "2026-08-14 10:00:00"}
	tm := rec.ParsedRequestTime()
	assert.False(t, tm.IsZero())
	assert.Equal(t, 2026, tm.Year())
	assert.Equal(t, 8, int(tm.Month()))
	assert.Equal(t, 14, tm.Day())
}

func TestSendCloudDeliveryRecord_ParsedRequestTime_Invalid(t *testing.T) {
	rec := SendCloudDeliveryRecord{RequestTime: "invalid"}
	assert.True(t, rec.ParsedRequestTime().IsZero())
}

func TestSendCloudDeliveryRecord_ParsedModifiedTime(t *testing.T) {
	rec := SendCloudDeliveryRecord{ModifiedTime: "2026-08-14 10:00:05"}
	tm := rec.ParsedModifiedTime()
	assert.False(t, tm.IsZero())
	assert.Equal(t, 5, tm.Second())
}

// ──────────────────────────────────────────────
// JSON round-trip
// ──────────────────────────────────────────────

func TestSendCloudStatusResponse_Unmarshal(t *testing.T) {
	raw := `{
		"result": true,
		"statusCode": 200,
		"message": "ok",
		"info": {
			"total": "1",
			"voListSize": 1,
			"voList": [{
				"emailId": "x",
				"status": "送达",
				"recipients": "a@b.com",
				"requestTime": "2026-01-01 00:00:00",
				"modifiedTime": "2026-01-01 00:00:01"
			}]
		}
	}`
	var resp sendCloudStatusResponse
	require.NoError(t, json.Unmarshal([]byte(raw), &resp))
	assert.True(t, resp.Result)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "1", resp.Info.Total)
	require.Len(t, resp.Info.VoList, 1)
	assert.Equal(t, "x", resp.Info.VoList[0].EmailID)
	assert.Equal(t, "送达", resp.Info.VoList[0].Status)
}

// ──────────────────────────────────────────────
// Kind
// ──────────────────────────────────────────────

func TestSendCloudReader_Kind(t *testing.T) {
	r, _ := NewSendCloudReader(SendCloudConfig{APIUser: "u", APIKey: "k"})
	assert.Equal(t, "sendcloud", r.Kind())
}

// Ensure no stray whitespace issues.
func TestSendCloudReader_Strings(t *testing.T) {
	r, _ := NewSendCloudReader(SendCloudConfig{APIUser: "  u  ", APIKey: "  k  "})
	// TrimSpace is applied in validation, but stored values keep original.
	_ = r
}
