package render

import (
	"bytes"
	"html/template"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	builder "github.com/benpate/exp-builder"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Profile struct {
	layout *model.Layout
	user   *model.User
	Common
}

func NewProfile(factory Factory, ctx *steranko.Context, user *model.User, actionID string) (Profile, error) {

	layout := factory.Layout().Profile()

	// Verify the requested action
	action := layout.Action(actionID)

	if action == nil {
		return Profile{}, derp.NewBadRequestError("render.NewProfile", "Invalid action", actionID)
	}

	return Profile{
		layout: layout,
		user:   user,
		Common: NewCommon(factory, ctx, action, actionID),
	}, nil
}

/*******************************************
 * RENDERER INTERFACE
 *******************************************/

// Render generates the string value for this Profile
func (w Profile) Render() (template.HTML, error) {

	var buffer bytes.Buffer

	// Execute step (write HTML to buffer, update context)
	if err := Pipeline(w.action.Steps).Get(w.factory(), &w, &buffer); err != nil {
		return "", derp.Report(derp.Wrap(err, "render.Profile.Render", "Error generating HTML"))

	}

	// Success!
	return template.HTML(buffer.String()), nil
}

// View executes a separate view for this Profile
func (w Profile) View(actionID string) (template.HTML, error) {

	renderer, err := NewProfile(w.factory(), w.ctx, w.user, actionID)

	if err != nil {
		return template.HTML(""), derp.Wrap(err, "render.Profile.View", "Error creating Profile renderer")
	}

	return renderer.Render()
}

func (w Profile) TopLevelID() string {

	if w.UserID() == w.Common.UserID().Hex() {

		if w.actionID == "inbox" {
			return "inbox"
		}
		return "profile"
	}

	return ""
}

func (w Profile) PageTitle() string {

	if w.UserID() == w.Common.UserID().Hex() {

		if w.actionID == "inbox" {
			return "Inbox"
		}
		return "Profile"
	}

	return ""
}

func (w Profile) Token() string {
	return "users"
}

func (w Profile) object() data.Object {
	return w.user
}

func (w Profile) objectID() primitive.ObjectID {
	return w.user.UserID
}

func (w Profile) schema() schema.Schema {
	return w.user.Schema()
}

func (w Profile) service() ModelService {
	return w.f.User()
}

func (w Profile) executeTemplate(writer io.Writer, name string, data any) error {
	return w.layout.HTMLTemplate.ExecuteTemplate(writer, name, data)
}

// UserCan returns TRUE if this Request is authorized to access the requested view
func (w Profile) UserCan(actionID string) bool {

	action := w.layout.Action(actionID)

	if action == nil {
		return false
	}

	authorization := w.authorization()

	return action.UserCan(w.user, &authorization)
}

/*******************************************
 * DATA ACCESSORS
 *******************************************/

func (w Profile) UserID() string {
	return w.user.UserID.Hex()
}

func (w Profile) DisplayName() string {
	return w.user.DisplayName
}

func (w Profile) Description() string {
	return w.user.Description
}

func (w Profile) ImageURL() string {
	return w.user.ImageURL
}

func (w Profile) Inbox() QueryBuilder {
	factory := w.factory()
	context := w.context()
	streamService := w.factory().Stream()
	criteria := exp.Equal("parentId", w.user.InboxID)

	return NewQueryBuilder(factory, context, streamService, criteria)
}

func (w Profile) Outbox() QueryBuilder {
	factory := w.factory()
	context := w.context()
	streamService := w.factory().Stream()
	criteria := exp.Equal("parentId", w.user.OutboxID)

	return NewQueryBuilder(factory, context, streamService, criteria)
}

/*******************************************
 * QUERY BUILDERS
 *******************************************/

func (w Profile) Profiles() *QueryBuilder {

	query := builder.NewBuilder().
		String("displayName").
		ObjectID("groupId")

	criteria := exp.And(
		query.Evaluate(w.ctx.Request().URL.Query()),
		exp.Equal("journal.deleteDate", 0),
	)

	result := NewQueryBuilder(w.factory(), w.ctx, w.factory().User(), criteria)

	return &result
}
