package opencode

import (
	"testing"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
)

func TestParserTextEmitsAssistantDelta(t *testing.T) {
	parser := NewParser(agent.RunRequest{ConversationID: "conv-1"})
	events, err := parser.ParseLine([]byte(`{"type":"text","sessionID":"sess_abc","part":{"id":"msg_1","type":"text","text":"Hello world"}}`))
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2 (session + text)", len(events))
	}
	if events[0].Type != agent.EventSessionUpdated || events[0].SessionID != "sess_abc" {
		t.Fatalf("first event = %#v, want session update", events[0])
	}
	text := events[1]
	if text.Type != agent.EventAssistantTextDelta || text.ItemKind != agent.ItemMessage || text.Text != "Hello world" {
		t.Fatalf("text event = %#v", text)
	}
	if text.Provider != agent.ProviderOpenCode || text.ConversationID != "conv-1" {
		t.Fatalf("text event missing context: %#v", text)
	}
}

func TestParserToolUseEmitsStartedAndCompleted(t *testing.T) {
	parser := NewParser(agent.RunRequest{})
	events, err := parser.ParseLine([]byte(`{"type":"tool_use","sessionID":"sess_abc","part":{"id":"tool_1","type":"tool","tool":"bash","state":{"status":"completed","input":{"command":"ls"},"output":"src"}}}`))
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("events = %d, want 3 (session + started + completed)", len(events))
	}
	started := events[1]
	if started.Type != agent.EventToolStarted || started.ToolName != "bash" || started.ItemID != "tool_1" {
		t.Fatalf("started = %#v", started)
	}
	if string(started.Input) != `{"command":"ls"}` {
		t.Fatalf("started input = %s", started.Input)
	}
	completed := events[2]
	if completed.Type != agent.EventToolCompleted || completed.Output != "src" || completed.IsError {
		t.Fatalf("completed = %#v", completed)
	}
}

func TestParserToolErrorMarksIsError(t *testing.T) {
	parser := NewParser(agent.RunRequest{})
	events, err := parser.ParseLine([]byte(`{"type":"tool_use","part":{"id":"tool_2","type":"tool","tool":"bash","state":{"status":"error","input":{},"output":"boom","error":"exit code 1"}}}`))
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	var completed agent.Event
	for _, ev := range events {
		if ev.Type == agent.EventToolCompleted {
			completed = ev
		}
	}
	if completed.Output != "exit code 1" || !completed.IsError {
		t.Fatalf("completed = %#v, want error output", completed)
	}
}

func TestParserReasoningEmitsDelta(t *testing.T) {
	parser := NewParser(agent.RunRequest{})
	events, err := parser.ParseLine([]byte(`{"type":"reasoning","part":{"id":"r_1","type":"reasoning","text":"thinking..."}}`))
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if len(events) != 1 || events[0].Type != agent.EventReasoningDelta || events[0].ItemKind != agent.ItemReasoning {
		t.Fatalf("events = %#v", events)
	}
}

func TestParserErrorEmitsErrorEvent(t *testing.T) {
	parser := NewParser(agent.RunRequest{})
	events, err := parser.ParseLine([]byte(`{"type":"error","error":{"name":"APIError","data":{"message":"model not found"}}}`))
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if len(events) != 1 || events[0].Type != agent.EventError || !events[0].IsError || events[0].Message != "model not found" {
		t.Fatalf("events = %#v", events)
	}
}

func TestParserIgnoresUnknownAndEmpty(t *testing.T) {
	parser := NewParser(agent.RunRequest{})
	for _, line := range []string{
		`{"type":"step_start","part":{"id":"s1"}}`,
		`{"type":"text","part":{"id":"t1","type":"text","text":"   "}}`,
		`{"type":"unknown","foo":"bar"}`,
	} {
		events, err := parser.ParseLine([]byte(line))
		if err != nil {
			t.Fatalf("ParseLine(%s): %v", line, err)
		}
		if len(events) != 0 {
			t.Fatalf("ParseLine(%s) = %d events, want 0", line, len(events))
		}
	}
}

func TestParserSessionEmittedOnce(t *testing.T) {
	parser := NewParser(agent.RunRequest{})
	first, err := parser.ParseLine([]byte(`{"type":"text","sessionID":"sess_x","part":{"id":"m1","type":"text","text":"a"}}`))
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if len(first) != 2 || first[0].Type != agent.EventSessionUpdated {
		t.Fatalf("first = %#v", first)
	}
	second, err := parser.ParseLine([]byte(`{"type":"text","sessionID":"sess_x","part":{"id":"m2","type":"text","text":"b"}}`))
	if err != nil {
		t.Fatalf("ParseLine: %v", err)
	}
	if len(second) != 1 || second[0].Type != agent.EventAssistantTextDelta {
		t.Fatalf("second = %#v, want only text delta", second)
	}
}
