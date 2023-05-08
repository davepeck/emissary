package service

import (
	"context"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Response defines a service that can send and receive response data
type Response struct {
	collection    data.Collection
	blockService  *Block
	inboxService  *Inbox
	outboxService *Outbox
	host          string
}

// NewResponse returns a fully initialized Response service
func NewResponse() Response {
	return Response{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Response) Refresh(collection data.Collection, blockService *Block, inboxService *Inbox, outboxService *Outbox, host string) {
	service.collection = collection
	service.blockService = blockService
	service.inboxService = inboxService
	service.outboxService = outboxService
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *Response) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

// Query returns a slice containing all of the Responses that match the provided criteria
func (service *Response) Query(criteria exp.Expression, options ...option.Option) ([]model.Response, error) {
	result := make([]model.Response, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Responses that match the provided criteria
func (service *Response) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Response from the database
func (service *Response) Load(criteria exp.Expression, response *model.Response) error {

	if err := service.collection.Load(notDeleted(criteria), response); err != nil {
		return derp.Wrap(err, "service.Response.Load", "Error loading Response", criteria)
	}

	return nil
}

// Save adds/updates an Response in the database
func (service *Response) Save(response *model.Response, note string) error {

	// Clean the value before saving
	if err := service.Schema().Clean(response); err != nil {
		return derp.Wrap(err, "service.Response.Save", "Error cleaning Response", response)
	}

	// Responses from Local Actor should be published to the Outbox
	if response.FromLocalActor() {
		if err := service.outboxService.Publish(response.Actor.InternalID, response.ResponseID, response.GetJSONLD()); err != nil {
			return derp.Wrap(err, "service.Response.Save", "Error publishing Response", response)
		}
	}

	// Response from Remote Actors should be filtered
	if response.FromRemoteActor() {
		// RULE: Filter Responses that are blocked
		if err := service.blockService.FilterResponse(response); err != nil {
			return derp.Wrap(err, "service.Response.Save", "Error filtering Response", response)
		}
	}

	// Recalculate statistics for the Message affected by this Response.
	if err := service.CalculateMessageStatistics(response); err != nil {
		return derp.Wrap(err, "service.Response.Save", "Error calculating message statistics", response)
	}

	// Save the value to the database
	if err := service.collection.Save(response, note); err != nil {
		return derp.Wrap(err, "service.Response.Save", "Error saving Response", response, note)
	}

	return nil
}

// Delete removes an Response from the database (virtual delete)
func (service *Response) Delete(response *model.Response, note string) error {

	criteria := exp.Equal("_id", response.ResponseID)

	// Delete this Response
	if err := service.collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, "service.Response.Delete", "Error deleting Response", criteria)
	}

	if response.FromLocalActor() {

		// Create an "Undo" activity
		activity := response.GetJSONLD()
		activity["type"] = vocab.ActivityTypeUndo
		activity["object"] = response.GetJSONLD()

		// Send the "Undo" activity
		if err := service.outboxService.UnPublish(response.Actor.InternalID, response.ResponseID, activity); err != nil {
			return derp.Wrap(err, "service.Response.Save", "Error publishing Response", response)
		}
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Response) ObjectType() string {
	return "Response"
}

// New returns a fully initialized model.Group as a data.Object.
func (service *Response) ObjectNew() data.Object {
	result := model.NewResponse()
	return &result
}

func (service *Response) ObjectID(object data.Object) primitive.ObjectID {

	if response, ok := object.(*model.Response); ok {
		return response.ResponseID
	}

	return primitive.NilObjectID
}

func (service *Response) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Response) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Response) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewResponse()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Response) ObjectSave(object data.Object, comment string) error {
	if response, ok := object.(*model.Response); ok {
		return service.Save(response, comment)
	}
	return derp.NewInternalError("service.Response.ObjectSave", "Invalid Object Type", object)
}

func (service *Response) ObjectDelete(object data.Object, comment string) error {
	if response, ok := object.(*model.Response); ok {
		return service.Delete(response, comment)
	}
	return derp.NewInternalError("service.Response.ObjectDelete", "Invalid Object Type", object)
}

func (service *Response) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Response", "Not Authorized")
}

func (service *Response) Schema() schema.Schema {
	return schema.New(model.ResponseSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Response) LoadOrCreate(userID primitive.ObjectID, objectID primitive.ObjectID) (model.Response, error) {

	response := model.NewResponse()

	err := service.LoadByObjectID(userID, objectID, &response)

	if err == nil {
		return response, nil
	}

	if derp.NotFound(err) {
		response.ObjectID = objectID
		response.Actor.InternalID = userID
		return response, nil
	}

	return response, derp.Wrap(err, "service.Response.LoadOrCreate", "Error loading Response", userID, objectID)
}

func (service *Response) LoadByID(userID primitive.ObjectID, responseID primitive.ObjectID, response *model.Response) error {

	criteria := exp.Equal("_id", responseID).
		AndEqual("actor.internalId", userID)

	if err := service.Load(criteria, response); err != nil {
		return derp.Wrap(err, "service.Response.LoadByID", "Error loading Response", responseID)
	}

	return nil
}

func (service *Response) LoadByObjectID(userID primitive.ObjectID, objectID primitive.ObjectID, response *model.Response) error {

	criteria := exp.Equal("objectId", objectID).
		AndEqual("actor.internalId", userID)

	if err := service.Load(criteria, response); err != nil {
		return derp.Wrap(err, "service.Response.LoadByID", "Error loading Response", userID, objectID)
	}

	return nil
}

func (service *Response) QueryByObjectID(objectID primitive.ObjectID) ([]model.Response, error) {
	criteria := exp.Equal("objectId", objectID)
	return service.Query(criteria)
}

func (service *Response) CalculateMessageStatistics(response *model.Response) error {

	spew.Dump(response)
	objectID := response.ObjectID

	if objectID.IsZero() {
		return nil
	}

	message := model.NewMessage()

	if err := service.inboxService.loadByID(objectID, &message); err != nil {
		return derp.Wrap(err, "service.Response.CalculateMessageStatistics", "Error loading Message", objectID)
	}

	if err := queries.CalculateMessageStatistics(context.TODO(), service.collection, &message); err != nil {
		return derp.Wrap(err, "service.Response.CalculateMessageStatistics", "Error calculating message statistics", message)
	}

	if err := service.inboxService.Save(&message, "Updated Response Totals"); err != nil {
		return derp.Wrap(err, "service.Response.CalculateMessageStatistics", "Error saving Message", message)
	}

	return nil
}
