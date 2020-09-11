package service

import (
	"github.com/benpate/data"
	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionStreamSource is the database collection where Sources are stored
const CollectionStreamSource = "StreamSource"

// StreamSource manages all interactions with the StreamSource collection
type StreamSource struct {
	factory    *Factory
	collection data.Collection
}

// New creates a newly initialized StreamSource that is ready to use
func (service StreamSource) New() *model.StreamSource {

	return &model.StreamSource{
		StreamSourceID: primitive.NewObjectID(),
	}
}

// List returns an iterator containing all of the StreamSources who match the provided criteria
func (service StreamSource) List(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an StreamSource from the database
func (service StreamSource) Load(criteria expression.Expression) (*model.StreamSource, *derp.Error) {

	account := service.New()

	if err := service.collection.Load(criteria, account); err != nil {
		return nil, derp.Wrap(err, "service.StreamSource", "Error loading StreamSource", criteria)
	}

	return account, nil
}

// Save adds/updates an StreamSource in the database
func (service StreamSource) Save(account *model.StreamSource, note string) *derp.Error {

	if err := service.collection.Save(account, note); err != nil {
		return derp.Wrap(err, "service.StreamSource", "Error saving StreamSource", account, note)
	}

	return nil
}

// Delete removes an StreamSource from the database (virtual delete)
func (service StreamSource) Delete(account *model.StreamSource, note string) *derp.Error {

	if err := service.collection.Delete(account, note); err != nil {
		return derp.Wrap(err, "service.StreamSource", "Error deleting StreamSource", account, note)
	}

	return nil
}

//// GENERIC FUNCTIONS //////////////////

// NewObject wraps the `New` method as a generic Object
func (service StreamSource) NewObject() data.Object {
	return service.New()
}

// ListObjects wraps the `List` method as a generic Object
func (service StreamSource) ListObjects(criteria expression.Expression, options ...option.Option) (data.Iterator, *derp.Error) {
	return service.List(criteria, options...)
}

// LoadObject wraps the `Load` method as a generic Object
func (service StreamSource) LoadObject(criteria expression.Expression) (data.Object, *derp.Error) {
	return service.Load(criteria)
}

// SaveObject wraps the `Save` method as a generic Object
func (service StreamSource) SaveObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.StreamSource); ok {
		return service.Save(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.StreamSource", "Object is not a model.StreamSource", object, note)
}

// DeleteObject wraps the `Delete` method as a generic Object
func (service StreamSource) DeleteObject(object data.Object, note string) *derp.Error {

	if object, ok := object.(*model.StreamSource); ok {
		return service.Delete(object, note)
	}

	// This should never happen.
	return derp.New(derp.CodeInternalError, "service.StreamSource", "Object is not a model.StreamSource", object, note)
}

/*
// Close cleans up the service and any outstanding connections.
func (service StreamSource) Close() {
	service.collection.Close()
}
*/

/// QUERIES //////////////////////////////////

// ListByMethod identifies all streamSources with a specific "Method" field
func (service StreamSource) ListByMethod(method model.StreamSourceMethod) (data.Iterator, *derp.Error) {

	criteria := expression.New("method", expression.OperatorEqual, method)

	return service.List(criteria)
}
