// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

import (
	"errors"
	"sync"
	"time"
)

// EventType identifies a speech state transition.
type EventType int

const (
	EventNone        EventType = iota
	EventSpeechStart           // speech segment started
	EventSpeechEnd             // speech segment ended
)

// Event is a speech state transition emitted by Streamer.
type Event struct {
	Type      EventType
	Timestamp time.Time
	// FrameIndex is the sequential frame number since the stream started.
	FrameIndex int
	// Probability is the speech probability at the transition frame.
	Probability float64
}

// String returns a human-readable description.
func (e Event) String() string {
	switch e.Type {
	case EventSpeechStart:
		return "speech_start"
	case EventSpeechEnd:
		return "speech_end"
	default:
		return "none"
	}
}

// Streamer wraps a Detector with a state machine that emits SpeechStart /
// SpeechEnd events. It implements hangover (debounce) so brief silence gaps
// within a speech segment don't cause false ends.
//
// Streamer is safe for concurrent use: ProcessFrame can be called from the
// audio goroutine while events are consumed from another goroutine via
// Events() channel.
type Streamer struct {
	detector Detector
	cfg      Config

	mu          sync.Mutex
	speechActive bool
	frameIndex   int
	speechFrames int // consecutive speech frames
	silenceFrames int // consecutive non-speech frames in hangover
	lastEvent    Event

	eventCh chan Event
	done    chan struct{}
}

// NewStreamer creates a streaming VAD state machine over the given detector.
// The config controls MinSpeechFrames (onset) and HangoverFrames (offset
// debounce).
func NewStreamer(detector Detector, cfg Config) *Streamer {
	cfg.validate()
	return &Streamer{
		detector: detector,
		cfg:      cfg,
		eventCh:  make(chan Event, 16),
		done:     make(chan struct{}),
	}
}

// ProcessFrame feeds a PCM16 LE frame to the detector and updates the state
// machine. Returns the FrameResult from the underlying detector.
//
// If the frame triggers a state transition, an Event is sent to the Events()
// channel (non-blocking; if the channel is full the event is dropped and
// the lastEvent field is still updated).
func (s *Streamer) ProcessFrame(pcm []byte) (FrameResult, error) {
	if s == nil || s.detector == nil {
		return FrameResult{}, errors.New("vad: nil streamer or detector")
	}

	result, err := s.detector.ProcessFrame(pcm)
	if err != nil {
		return result, err
	}

	s.mu.Lock()
	s.frameIndex++
	event := Event{
		Type:       EventNone,
		Timestamp:  result.Timestamp,
		FrameIndex: s.frameIndex,
		Probability: result.Probability,
	}

	if result.IsSpeech {
		s.silenceFrames = 0
		s.speechFrames++

		if !s.speechActive && s.speechFrames >= s.cfg.MinSpeechFrames {
			s.speechActive = true
			event.Type = EventSpeechStart
		}
	} else {
		s.speechFrames = 0
		if s.speechActive {
			s.silenceFrames++
			if s.silenceFrames >= s.cfg.HangoverFrames {
				s.speechActive = false
				s.silenceFrames = 0
				event.Type = EventSpeechEnd
			}
		}
	}

	if event.Type != EventNone {
		s.lastEvent = event
		s.mu.Unlock()
		// Non-blocking send; drop if consumer is slow.
		select {
		case s.eventCh <- event:
		default:
		}
	} else {
		s.mu.Unlock()
	}

	return result, nil
}

// Events returns a channel of speech state transitions.
// The channel is buffered; events are dropped if the buffer is full.
func (s *Streamer) Events() <-chan Event {
	return s.eventCh
}

// IsSpeech returns the current speech state (after hangover smoothing).
func (s *Streamer) IsSpeech() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.speechActive
}

// FrameIndex returns the total number of frames processed.
func (s *Streamer) FrameIndex() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frameIndex
}

// LastEvent returns the most recent transition event.
func (s *Streamer) LastEvent() Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastEvent
}

// Reset clears the state machine and underlying detector.
func (s *Streamer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.speechActive = false
	s.speechFrames = 0
	s.silenceFrames = 0
	s.frameIndex = 0
	s.lastEvent = Event{}
	if s.detector != nil {
		s.detector.Reset()
	}
}

// Close releases resources. After Close, the Events channel is closed.
func (s *Streamer) Close() error {
	if s == nil {
		return nil
	}
	select {
	case <-s.done:
		return nil
	default:
	}
	close(s.done)
	close(s.eventCh)
	if s.detector != nil {
		return s.detector.Close()
	}
	return nil
}

// Done returns a channel that is closed when Close is called.
func (s *Streamer) Done() <-chan struct{} {
	return s.done
}
