package render

import (
	"bytes"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// StepEditContent represents an action-step that can edit/update Container in a streamDraft.
type StepEditContent struct {
	Filename string
	Format   string
}

func (step StepEditContent) Get(renderer Renderer, buffer io.Writer) ExitCondition {

	if err := renderer.executeTemplate(buffer, step.Filename, renderer); err != nil {
		return ExitError(derp.Wrap(err, "render.StepEditContent.Get", "Error executing template"))
	}

	return nil
}

func (step StepEditContent) Post(renderer Renderer, _ io.Writer) ExitCondition {

	context := renderer.context()

	var rawContent string

	// Try to read the content from the request body
	switch step.Format {

	// EditorJS writes directly to the request body
	case model.ContentFormatEditorJS:
		var buffer bytes.Buffer

		if _, err := io.Copy(&buffer, context.Request().Body); err != nil {
			return ExitError(derp.Wrap(err, "render.StepEditContent.Post", "Error reading request data"))
		}

		rawContent = buffer.String()

	// All other types are a Form post
	default:

		body := mapof.NewAny()
		if err := context.Bind(&body); err != nil {
			return ExitError(derp.Wrap(err, "render.StepEditContent.Post", "Error parsing request data"))
		}

		rawContent, _ = body.GetStringOK("content")
	}

	// Create a new Content object from the request body
	factory := renderer.factory()
	contentService := factory.Content()
	content := contentService.New(step.Format, rawContent)

	// Put the content into the stream
	stream := renderer.object().(*model.Stream)
	stream.Content = content

	// Try to save the object back to the database
	if err := renderer.service().ObjectSave(stream, "Content edited"); err != nil {
		return ExitError(derp.Wrap(err, "render.StepEditContent.Post", "Error saving stream"))
	}

	// Success!
	return nil
}
