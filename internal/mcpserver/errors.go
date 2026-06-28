package mcpserver

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stackrail/incident-investigator/internal/runtime"
)

// toolErr formats a user-facing tool error. Callers return (toolErr(...), zero, nil)
// so MCP clients receive IsError=true rather than a transport failure.
func toolErr(format string, args ...any) error {
	return errors.New(fmt.Sprintf(format, args...))
}

func mapRuntimeError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, runtime.ErrSessionNotFound):
		return toolErr("session not found")
	case errors.Is(err, runtime.ErrSessionCompleted):
		return toolErr("investigation is already completed; start a new investigation instead")
	case errors.Is(err, runtime.ErrEmptyEvidence):
		return toolErr("at least one evidence item is required")
	case errors.Is(err, runtime.ErrInvalidEvidence):
		return toolErr("each evidence item must include a non-empty summary")
	case errors.Is(err, runtime.ErrInvalidTimeWindow):
		return toolErr("time_window end must not be before start")
	case errors.Is(err, runtime.ErrDuplicateEvidenceID):
		return toolErr("%s", err.Error())
	default:
		return toolErr("%s", strings.TrimSpace(err.Error()))
	}
}
