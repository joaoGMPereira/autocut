// Package api provides the HTTP router and SSE type aliases for the AutoCut server.
// SSEHub is defined in internal/hub to avoid import cycles.
package api

import (
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// SSEEvent is re-exported from the hub package for backward compatibility.
type SSEEvent = hub.SSEEvent

// SSEHub is re-exported from the hub package.
type SSEHub = hub.SSEHub

// NewSSEHub creates an initialised SSEHub. Delegates to hub.New().
func NewSSEHub() *SSEHub {
	return hub.New()
}
