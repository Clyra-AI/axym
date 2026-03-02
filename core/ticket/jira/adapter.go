package jira

import (
	"context"
	"sync"

	"github.com/Clyra-AI/axym/core/ticket"
)

type ScriptedAdapter struct {
	mu        sync.Mutex
	responses []ticket.Response
	index     int
}

func NewScripted(statusCodes []int) *ScriptedAdapter {
	responses := make([]ticket.Response, 0, len(statusCodes))
	for _, statusCode := range statusCodes {
		responses = append(responses, ticket.Response{StatusCode: statusCode})
	}
	if len(responses) == 0 {
		responses = append(responses, ticket.Response{StatusCode: 200})
	}
	return &ScriptedAdapter{responses: responses}
}

func (a *ScriptedAdapter) Name() string { return "jira" }

func (a *ScriptedAdapter) Attach(_ context.Context, _ ticket.AttachRequest) (ticket.Response, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.index >= len(a.responses) {
		return a.responses[len(a.responses)-1], nil
	}
	resp := a.responses[a.index]
	a.index++
	return resp, nil
}
