package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/path"
)

// StepSetThumbnail represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepSetThumbnail struct{}

func (step StepSetThumbnail) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

func (step StepSetThumbnail) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step StepSetThumbnail) Post(renderer Renderer) error {

	// Find best icon from attachments
	factory := renderer.factory()
	attachments, err := factory.Attachment().ListByObjectID(renderer.objectID())

	if err != nil {
		return derp.NewBadRequestError("render.StepSetThumbnail.Post", "Error listing attachments")
	}

	// Scan all attachments and use the first one that is an image.
	attachment := model.NewAttachment(renderer.objectID())
	for attachments.Next(&attachment) {

		if attachment.MimeCategory() == "image" {
			return path.Set(renderer.object(), "thumbnailImage", attachment.Filename)
		}
		attachment = model.NewAttachment(renderer.objectID())
	}

	// Fall through to here means we should look at body content (but not now)

	return nil
}
