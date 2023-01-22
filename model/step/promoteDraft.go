package step

import (
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/maps"
)

// StreamPromoteDraft represents a pipeline-step that can copy the Container from a StreamDraft into its corresponding Stream
type StreamPromoteDraft struct {
	StateID string
}

func NewStreamPromoteDraft(stepInfo maps.Map) (StreamPromoteDraft, error) {
	return StreamPromoteDraft{
		StateID: first.String(getValue(stepInfo.GetString("state")), "published"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step StreamPromoteDraft) AmStep() {}
