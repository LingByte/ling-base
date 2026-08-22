//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

// Package multimodal converts AG-UI multimodal content into internal model messages.
package multimodal

import (
	"encoding/base64"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/LingByte/ling-base/agentkit/codeexecutor"
	"github.com/LingByte/ling-base/agentkit/internal/fileref"
	compat "github.com/LingByte/ling-base/relay/compat"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/types"
)

const (
	// CustomEventNameUserMessage is the custom event name used to persist user input messages in the track stream.
	CustomEventNameUserMessage = "trpc-agent-go.user_message"
)

// UserMessageFromInputContents converts AG-UI multimodal input contents into a model user message.
func UserMessageFromInputContents(contents []types.InputContent) (compat.Message, error) {
	if len(contents) == 0 {
		return compat.Message{}, errors.New("input contents is empty")
	}
	message := compat.Message{
		Role: compat.RoleUser,
	}
	for _, part := range contents {
		contentPart, err := contentPartFromInputContent(part)
		if err != nil {
			return compat.Message{}, err
		}
		if contentPart == nil {
			continue
		}
		message.ContentParts = append(message.ContentParts, *contentPart)
	}
	if len(message.ContentParts) == 0 {
		return compat.Message{}, errors.New("no supported input contents")
	}
	return message, nil
}

func contentPartFromInputContent(part types.InputContent) (*compat.ContentPart, error) {
	switch part.Type {
	case types.InputContentTypeText:
		text := part.Text
		return &compat.ContentPart{
			Type: compat.ContentTypeText,
			Text: &text,
		}, nil
	case types.InputContentTypeBinary:
		return contentPartFromBinaryInput(part)
	case types.InputContentTypeImage,
		types.InputContentTypeAudio,
		types.InputContentTypeVideo,
		types.InputContentTypeDocument:
		return contentPartFromTypedInput(part)
	default:
		return nil, nil
	}
}

func contentPartFromTypedInput(part types.InputContent) (*compat.ContentPart, error) {
	if part.Source == nil {
		return nil, fmt.Errorf("%s input content requires source", part.Type)
	}

	sourceType := strings.ToLower(strings.TrimSpace(part.Source.Type))
	value := strings.TrimSpace(part.Source.Value)
	if value == "" {
		return nil, fmt.Errorf("%s input content source value is empty", part.Type)
	}
	mimeType := strings.ToLower(strings.TrimSpace(part.Source.MimeType))

	switch sourceType {
	case types.InputContentSourceTypeURL:
		return contentPartFromTypedURL(part, value, mimeType)
	case types.InputContentSourceTypeData:
		if mimeType == "" {
			return nil, fmt.Errorf("%s data source requires mime type", part.Type)
		}
		payload, err := decodeBase64Payload(value)
		if err != nil {
			return nil, fmt.Errorf("decode %s payload: %w", part.Type, err)
		}
		return contentPartFromTypedData(part, payload, mimeType)
	default:
		return nil, fmt.Errorf("unsupported %s input content source type %q", part.Type, part.Source.Type)
	}
}

func contentPartFromTypedURL(
	part types.InputContent,
	url string,
	mimeType string,
) (*compat.ContentPart, error) {
	switch part.Type {
	case types.InputContentTypeImage:
		return &compat.ContentPart{
			Type:  compat.ContentTypeImage,
			Image: &compat.Image{URL: url, Format: mimeType},
		}, nil
	case types.InputContentTypeAudio:
		return &compat.ContentPart{
			Type:  compat.ContentTypeAudio,
			Audio: &compat.Audio{URL: url, Format: mimeType},
		}, nil
	case types.InputContentTypeVideo:
		return &compat.ContentPart{
			Type:  compat.ContentTypeVideo,
			Video: &compat.Video{URL: url, Format: mimeType},
		}, nil
	default:
		return &compat.ContentPart{
			Type: compat.ContentTypeFile,
			File: &compat.File{
				URL:      url,
				MimeType: mimeType,
			},
		}, nil
	}
}

func contentPartFromTypedData(
	part types.InputContent,
	payload []byte,
	mimeType string,
) (*compat.ContentPart, error) {
	switch part.Type {
	case types.InputContentTypeImage:
		format, err := mediaSubtype(mimeType, "image/", part.Type)
		if err != nil {
			return nil, err
		}
		return &compat.ContentPart{
			Type:  compat.ContentTypeImage,
			Image: &compat.Image{Data: payload, Format: format},
		}, nil
	case types.InputContentTypeAudio:
		format, err := mediaSubtype(mimeType, "audio/", part.Type)
		if err != nil {
			return nil, err
		}
		return &compat.ContentPart{
			Type:  compat.ContentTypeAudio,
			Audio: &compat.Audio{Data: payload, Format: format},
		}, nil
	case types.InputContentTypeVideo:
		format, err := mediaSubtype(mimeType, "video/", part.Type)
		if err != nil {
			return nil, err
		}
		return &compat.ContentPart{
			Type:  compat.ContentTypeVideo,
			Video: &compat.Video{Data: payload, Format: format},
		}, nil
	default:
		return &compat.ContentPart{
			Type: compat.ContentTypeFile,
			File: &compat.File{
				Data:     payload,
				MimeType: mimeType,
			},
		}, nil
	}
}

func mediaSubtype(mimeType, prefix, contentType string) (string, error) {
	format, ok := strings.CutPrefix(mimeType, prefix)
	if !ok || format == "" {
		return "", fmt.Errorf("%s data source requires %s mime type", contentType, contentType)
	}
	return format, nil
}

func contentPartFromBinaryInput(part types.InputContent) (*compat.ContentPart, error) {
	binaryURL := strings.TrimSpace(part.URL)
	if part.ID == "" && binaryURL == "" && part.Data == "" {
		return nil, errors.New("binary input content requires at least one of id, url, or data")
	}
	mimeType := strings.ToLower(strings.TrimSpace(part.MimeType))
	if binaryURL != "" {
		if strings.HasPrefix(mimeType, "image/") {
			return &compat.ContentPart{
				Type: compat.ContentTypeImage,
				Image: &compat.Image{
					URL:    binaryURL,
					Format: mimeType,
				},
			}, nil
		}
		return &compat.ContentPart{
			Type: compat.ContentTypeFile,
			File: fileFromBinaryURL(part, mimeType, binaryURL),
		}, nil
	}
	if part.Data != "" {
		payload, err := decodeBase64Payload(part.Data)
		if err != nil {
			return nil, fmt.Errorf("decode binary payload: %w", err)
		}
		if format, ok := strings.CutPrefix(mimeType, "audio/"); ok && format != "" {
			return &compat.ContentPart{
				Type: compat.ContentTypeAudio,
				Audio: &compat.Audio{
					Data:   payload,
					Format: format,
				},
			}, nil
		}
		if format, ok := strings.CutPrefix(mimeType, "image/"); ok && format != "" {
			return &compat.ContentPart{
				Type: compat.ContentTypeImage,
				Image: &compat.Image{
					Data:   payload,
					Format: format,
				},
			}, nil
		}
		filename := part.Filename
		return &compat.ContentPart{
			Type: compat.ContentTypeFile,
			File: &compat.File{
				Name:     filename,
				Data:     payload,
				MimeType: mimeType,
			},
		}, nil
	}
	return &compat.ContentPart{
		Type: compat.ContentTypeFile,
		File: fileFromBinaryID(part),
	}, nil
}

func fileFromBinaryURL(part types.InputContent, mimeType, fileURL string) *compat.File {
	return &compat.File{
		Name:     strings.TrimSpace(part.Filename),
		URL:      fileURL,
		MimeType: mimeType,
	}
}

func fileFromBinaryID(part types.InputContent) *compat.File {
	if part.Type != types.InputContentTypeBinary {
		return nil
	}
	file := &compat.File{
		Name:   strings.TrimSpace(part.Filename),
		FileID: part.ID,
	}
	if file.Name == "" {
		file.Name = fileNameFromArtifactRef(part.ID)
	}
	return file
}

func fileNameFromArtifactRef(fileID string) string {
	s := strings.TrimSpace(fileID)
	if !strings.HasPrefix(s, fileref.ArtifactPrefix) {
		return ""
	}
	rest := strings.TrimPrefix(s, fileref.ArtifactPrefix)
	name, _, err := codeexecutor.ParseArtifactRef(rest)
	if err != nil {
		return ""
	}
	base := path.Base(strings.TrimSpace(name))
	if base == "." || base == "/" || base == ".." {
		return ""
	}
	return base
}

func decodeBase64Payload(payload string) ([]byte, error) {
	if strings.HasPrefix(payload, "data:") {
		comma := strings.IndexByte(payload, ',')
		if comma < 0 {
			return nil, errors.New("base64 data URL is missing comma separator")
		}
		header := strings.ToLower(payload[:comma])
		if !strings.Contains(header, ";base64") {
			return nil, errors.New("data URL is not base64-encoded")
		}
		payload = payload[comma+1:]
	}
	if payload == "" {
		return nil, errors.New("base64 payload is empty")
	}
	return base64.StdEncoding.DecodeString(payload)
}
