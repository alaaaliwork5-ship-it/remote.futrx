package opencode

// Parser converts `opencode run --format json` NDJSON events into normalized
// agent events. OpenCode emits one JSON object per line:
//
//	{"type":"text","sessionID":"sess_...","part":{"id":"...","type":"text","text":"..."}}
//	{"type":"reasoning","part":{"id":"...","type":"reasoning","text":"..."}}
//	{"type":"tool_use","part":{"id":"...","type":"tool","tool":"bash","state":{"status":"completed","input":{...},"output":"..."}}}
//	{"type":"error","error":{"name":"...","data":{"message":"..."}}}
//
// and exits when the session goes idle, so run completion is signaled by the
// provider once the process exits cleanly (there is no explicit "completed"
// event).

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/futrx-com/remote.futrx.com/internal/agent"
)

type Parser struct {
	req          agent.RunRequest
	sawSessionID string
}

func NewParser(req agent.RunRequest) *Parser {
	if req.Provider == "" {
		req.Provider = agent.ProviderOpenCode
	}
	return &Parser{req: req, sawSessionID: req.ResumeID}
}

type openCodeEvent struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionID,omitempty"`
	Part      openCodePart    `json:"part,omitempty"`
	Error     openCodeError   `json:"error,omitempty"`
	Raw       json.RawMessage `json:"-"`
}

func (e *openCodeEvent) UnmarshalJSON(data []byte) error {
	type alias openCodeEvent
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*e = openCodeEvent(a)
	e.Raw = append(json.RawMessage(nil), data...)
	return nil
}

type openCodePart struct {
	ID    string        `json:"id,omitempty"`
	Type  string        `json:"type,omitempty"`
	Tool  string        `json:"tool,omitempty"`
	Text  string        `json:"text,omitempty"`
	State openCodeState `json:"state,omitempty"`
}

type openCodeState struct {
	Status string          `json:"status,omitempty"`
	Input  json.RawMessage `json:"input,omitempty"`
	Output string          `json:"output,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type openCodeError struct {
	Name string `json:"name,omitempty"`
	Data struct {
		Message string `json:"message,omitempty"`
	} `json:"data,omitempty"`
}

func (p *Parser) ParseLine(line []byte) ([]agent.Event, error) {
	var raw openCodeEvent
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()
	events := make([]agent.Event, 0, 3)

	if raw.SessionID != "" && raw.SessionID != p.sawSessionID {
		p.sawSessionID = raw.SessionID
		events = append(events, p.event(now, agent.EventSessionUpdated, raw.Raw, func(ev *agent.Event) {
			ev.SessionID = raw.SessionID
		}))
	}

	switch raw.Type {
	case "text":
		text := strings.TrimSpace(raw.Part.Text)
		if text == "" {
			return events, nil
		}
		events = append(events, p.event(now, agent.EventAssistantTextDelta, raw.Raw, func(ev *agent.Event) {
			ev.ItemKind = agent.ItemMessage
			ev.ItemID = raw.Part.ID
			ev.Text = text
		}))

	case "reasoning":
		text := strings.TrimSpace(raw.Part.Text)
		if text == "" {
			return events, nil
		}
		events = append(events, p.event(now, agent.EventReasoningDelta, raw.Raw, func(ev *agent.Event) {
			ev.ItemKind = agent.ItemReasoning
			ev.ItemID = raw.Part.ID
			ev.Text = text
		}))

	case "tool_use":
		toolName := strings.TrimSpace(raw.Part.Tool)
		if toolName == "" {
			toolName = "OpenCodeTool"
		}
		isError := raw.Part.State.Status == "error"
		events = append(events,
			p.event(now, agent.EventToolStarted, raw.Raw, func(ev *agent.Event) {
				ev.ItemKind = agent.ItemToolCall
				ev.ItemID = raw.Part.ID
				ev.ToolName = toolName
				if len(raw.Part.State.Input) > 0 {
					ev.Input = raw.Part.State.Input
				}
			}),
			p.event(now, agent.EventToolCompleted, raw.Raw, func(ev *agent.Event) {
				ev.ItemKind = agent.ItemToolCall
				ev.ItemID = raw.Part.ID
				ev.Output = raw.Part.State.Output
				ev.IsError = isError
				if isError && raw.Part.State.Error != "" {
					ev.Output = raw.Part.State.Error
				}
			}),
		)

	case "error":
		message := strings.TrimSpace(raw.Error.Data.Message)
		if message == "" {
			message = strings.TrimSpace(raw.Error.Name)
		}
		if message == "" {
			message = "OpenCode run error"
		}
		events = append(events, p.event(now, agent.EventError, raw.Raw, func(ev *agent.Event) {
			ev.Message = message
			ev.IsError = true
		}))
	}

	return events, nil
}

func (p *Parser) event(now int64, type_ agent.EventType, raw json.RawMessage, fn func(*agent.Event)) agent.Event {
	ev := agent.Event{
		T:              now,
		Type:           type_,
		Provider:       agent.ProviderOpenCode,
		ConversationID: p.req.ConversationID,
		Raw:            raw,
	}
	if fn != nil {
		fn(&ev)
	}
	return ev
}
