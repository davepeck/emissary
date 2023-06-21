package render

import (
	"encoding/json"
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/labstack/echo/v4"
)

type PipelineStatus struct {
	StatusCode  int          // HTTP Status Code to be returned
	ContentType string       // If present, then this option sets the content-type header
	Headers     mapof.String // Map of header values to be applied to the response
	Events      mapof.String // Map of events to trigger on the client (via HX-Trigger)
	FullPage    bool         // If true, then this result represents the entire page of content, and should not be wrapped in the global template
	Halt        bool         // If true, then this pipeline should halt execution
	Error       error        // If present, then there was an error rendering this page
}

func NewPipelineStatus() PipelineStatus {
	return PipelineStatus{
		Headers: mapof.NewString(),
		Events:  mapof.NewString(),
	}
}

func (status PipelineStatus) GetContentType() string {

	if status.ContentType != "" {
		return status.ContentType
	}

	return "text/html"
}

func (status PipelineStatus) GetStatusCode() int {

	if status.StatusCode != 0 {
		return status.StatusCode
	}

	if status.Error != nil {
		return derp.ErrorCode(status.Error)
	}

	return http.StatusOK
}

func (status PipelineStatus) Apply(ctx echo.Context) {

	response := ctx.Response()
	header := response.Header()
	header.Set("Content-Type", status.GetContentType())

	// Copy HT-Trigger events into response
	if len(status.Events) > 0 {
		if hxTrigger, err := json.Marshal(status.Events); err == nil {
			header.Set("HX-Trigger", string(hxTrigger))
		}
	}

	// Copy OTHER headers into response
	for name, value := range status.Headers {
		header.Set(name, value)
	}
}

// Merge combines two PipelineStatus objects into one.
func (status *PipelineStatus) Merge(newStatus PipelineStatus) {

	// Copy bools into the result
	status.FullPage = newStatus.FullPage || status.FullPage
	status.Halt = newStatus.Halt || status.Halt

	// Copy Content Type into the result
	if newStatus.ContentType != "" {
		status.ContentType = newStatus.ContentType
	}

	// Copy Status Code into the result
	if newStatus.StatusCode != 0 {
		status.StatusCode = newStatus.StatusCode
	}

	// Copy HTTP headers into the result
	for name, value := range newStatus.Headers {
		if _, ok := status.Headers[name]; !ok {
			status.Headers[name] = value
		}
	}

	// Copy HX-Trigger headers into the result
	for name, value := range newStatus.Events {
		if _, ok := status.Events[name]; !ok {
			status.Events[name] = value
		}
	}

	// Copy Error value into the result
	if newStatus.Error != nil {
		status.Error = newStatus.Error
	}
}

type ExitCondition func(*PipelineStatus)

func Exit() ExitCondition {
	return func(_ *PipelineStatus) {}
}

// ExitHalt sets the Halt flag on the PipelineStatus object
func ExitHalt() ExitCondition {
	return func(status *PipelineStatus) {
		status.Halt = true
	}
}

// ExitError sets the Error value on the PipelineStatus object
func ExitError(err error) ExitCondition {
	return func(status *PipelineStatus) {
		status.Error = err
		status.Halt = true
	}
}

// ExitFullPage sets the FullPage flag on the PipelineStatus object
func ExitFullPage() ExitCondition {
	return func(status *PipelineStatus) {
		status.FullPage = true
	}
}

// ExitContentType sets the content-type header for the PipelineStatus object
func ExitContentType(contentType string) ExitCondition {
	return func(status *PipelineStatus) {
		status.ContentType = contentType
		status.FullPage = true
	}
}

// ExitWithStatus takes a new PipelineStatus object, and merges it into the existing PipelineStatus object.
func ExitWithStatus(newStatus PipelineStatus) ExitCondition {
	return func(oldStatus *PipelineStatus) {
		oldStatus.Merge(newStatus)
	}
}

func ExitWithEvent(name string, value string) ExitCondition {
	return func(status *PipelineStatus) {
		status.Events[name] = value
	}
}

// AsFullPage adds an HX-Trigger event to the PipelineStatus object
func (exit ExitCondition) AsFullPage() ExitCondition {
	return func(status *PipelineStatus) {
		if exit != nil {
			exit(status)
		}
		status.FullPage = true
	}
}

// WithHeader adds an HX-Trigger event to the PipelineStatus object
func (exit ExitCondition) WithHeader(name string, value string) ExitCondition {
	return func(status *PipelineStatus) {
		if exit != nil {
			exit(status)
		}
		status.Headers[name] = value
	}
}

// WithEvent adds an HX-Trigger event to the PipelineStatus object
func (exit ExitCondition) WithEvent(name string, value string) ExitCondition {
	return func(status *PipelineStatus) {
		if exit != nil {
			exit(status)
		}
		status.Events[name] = value
	}
}

func (exit ExitCondition) WithStatusCode(statusCode int) ExitCondition {
	return func(status *PipelineStatus) {
		if exit != nil {
			exit(status)
		}
		status.StatusCode = statusCode
	}
}

func (exit ExitCondition) WithContentType(contentType string) ExitCondition {
	return func(status *PipelineStatus) {
		exit(status)
		status.ContentType = contentType
	}
}
