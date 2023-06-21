package render

import (
	"io"
)

// StepRefreshPage represents an action-step that forwards the user to a new page.
type StepRefreshPage struct{}

func (step StepRefreshPage) Get(renderer Renderer, _ io.Writer) ExitCondition {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepRefreshPage) Post(renderer Renderer, _ io.Writer) ExitCondition {
	return Exit().
		WithEvent("closeModal", "true").
		WithEvent("refreshPage", "true")
}
