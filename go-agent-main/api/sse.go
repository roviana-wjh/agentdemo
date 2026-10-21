package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	SSEEventStart    = "start"
	SSEEventDelta    = "delta"
	SSEEventCitation = "citation"
	SSEEventTool     = "tool"
	SSEEventRoute    = "route"
	SSEEventDone     = "done"
	SSEEventError    = "error"
)

type SSEError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SSEPayload is the stable JSON envelope shared by all Agent streaming APIs.
type SSEPayload struct {
	Type      string    `json:"type"`
	RequestID string    `json:"request_id"`
	SessionID string    `json:"session_id,omitempty"`
	Content   string    `json:"content,omitempty"`
	Data      any       `json:"data,omitempty"`
	Error     *SSEError `json:"error,omitempty"`
}

type SSEWriter struct {
	ctx       *gin.Context
	flusher   http.Flusher
	requestID string
	sessionID string
	started   bool
}

func NewSSEWriter(c *gin.Context, requestID, sessionID string) (*SSEWriter, error) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}
	if requestID == "" {
		requestID = uuid.NewString()
	}
	return &SSEWriter{
		ctx:       c,
		flusher:   flusher,
		requestID: requestID,
		sessionID: sessionID,
	}, nil
}

func (w *SSEWriter) Start() {
	if w.started {
		return
	}
	w.ctx.Header("Content-Type", "text/event-stream; charset=utf-8")
	w.ctx.Header("Cache-Control", "no-cache")
	w.ctx.Header("Connection", "keep-alive")
	w.ctx.Header("X-Accel-Buffering", "no")
	w.ctx.Header("X-Request-ID", w.requestID)
	w.ctx.Status(http.StatusOK)
	w.started = true
	w.Event(SSEEventStart, nil)
}

func (w *SSEWriter) Event(event string, data any) {
	payload := SSEPayload{
		Type:      event,
		RequestID: w.requestID,
		SessionID: w.sessionID,
		Data:      data,
	}
	w.ctx.SSEvent(event, payload)
	w.flusher.Flush()
}

func (w *SSEWriter) Delta(content string) {
	payload := SSEPayload{
		Type:      SSEEventDelta,
		RequestID: w.requestID,
		SessionID: w.sessionID,
		Content:   content,
	}
	w.ctx.SSEvent(SSEEventDelta, payload)
	w.flusher.Flush()
}

func (w *SSEWriter) Citation(data any) {
	w.Event(SSEEventCitation, data)
}

func (w *SSEWriter) Tool(data any) {
	w.Event(SSEEventTool, data)
}

func (w *SSEWriter) Done() {
	w.Event(SSEEventDone, nil)
}

func (w *SSEWriter) Error(code, message string) {
	payload := SSEPayload{
		Type:      SSEEventError,
		RequestID: w.requestID,
		SessionID: w.sessionID,
		Error: &SSEError{
			Code:    code,
			Message: message,
		},
	}
	w.ctx.SSEvent(SSEEventError, payload)
	w.flusher.Flush()
}
