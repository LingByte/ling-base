package recognizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// SpeechRecognitionEngine is the legacy alias for Engine (LingEchoX compatibility).
type SpeechRecognitionEngine = Engine

// Legacy callback type aliases (LingEchoX compatibility).
type (
	SpeechRecognitionResult = ResultFunc
	RecognitionError        = ErrorFunc
)