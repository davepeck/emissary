package handler

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/render"
	"github.com/EmissarySocial/emissary/server"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/html"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetFollowing displays an edit for for a specific following
func GetFollowing(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.GetFollowing"

	return func(ctx echo.Context) error {

		// Load all pre-requisites
		factory, following, userID, followingID, err := following_common(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading following", userID, followingID)
		}

		// Create a new form
		folderService := factory.Folder()
		folders := following_folderOptions(folderService, userID)
		form := following_getForm(folders)

		if followingID == "new" {
			form.Element.Label = "Follow a Person or Website"
		} else {
			form.Element.Label = "Edit Follow Settings"
		}

		html, err := form.Editor(following, nil)

		if err != nil {
			return derp.Wrap(err, location, "Error creating form editor", nil)
		}

		// Wrap the form as a modal dialog (with submit buttons)
		html = render.WrapModalForm(ctx.Response(), "/@me/pub/following/"+followingID, html)

		// Done.
		return ctx.HTML(http.StatusOK, html)
	}
}

func PostFollowing(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.PostFollowing"

	return func(ctx echo.Context) error {

		var transaction struct {
			URL           string `form:"url"`
			FolderID      string `form:"folderId"`
			PollDuration  int    `form:"pollDuration"`
			PurgeDuration int    `form:"purgeDuration"`
		}

		// Load all pre-requisites
		factory, following, userID, followingID, err := following_common(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading following", userID, followingID)
		}

		// Collect data from the form POST
		if err := ctx.Bind(&transaction); err != nil {
			return derp.Wrap(err, location, "Error reading form data", nil)
		}

		following.URL = transaction.URL
		following.PollDuration = transaction.PollDuration
		following.PurgeDuration = transaction.PurgeDuration

		if folderID, err := primitive.ObjectIDFromHex(transaction.FolderID); err == nil {
			following.FolderID = folderID
		} else {
			following.FolderID = primitive.NilObjectID
		}

		// Save the following to the database
		if err := factory.Following().Save(&following, "Updated by User"); err != nil {
			return derp.Wrap(err, location, "Error saving following", following)
		}

		// Close the Modal Dialog and return
		render.CloseModal(ctx, "")
		return ctx.NoContent(http.StatusOK)
	}
}

func GetDeleteFollowing(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.DeleteFollowing"

	return func(ctx echo.Context) error {

		_, following, userID, followingID, err := following_common(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading following", userID, followingID)
		}

		b := html.New()

		b.H2().InnerHTML("Stop Following?").Close()
		b.Div().Class("space-below").InnerHTML(following.Label).Close()
		b.Div().Class("space-below").InnerHTML(following.URL).Close()

		b.Button().Class("warning").
			Attr("hx-post", "/@me/pub/following/"+following.FollowingID.Hex()+"/delete").
			Attr("hx-swap", "none").
			InnerHTML("Delete Following").
			Close()

		b.Button().Script("on click trigger closeModal").InnerHTML("Cancel").Close()
		b.CloseAll()

		result := render.WrapModal(ctx.Response(), b.String())
		io.WriteString(ctx.Response(), result)
		return nil
	}
}

func PostDeleteFollowing(serverFactory *server.Factory) echo.HandlerFunc {

	const location = "handler.DeleteFollowing"

	return func(ctx echo.Context) error {

		factory, following, userID, followingID, err := following_common(serverFactory, ctx)

		if err != nil {
			return derp.Wrap(err, location, "Error loading following", userID, followingID)
		}

		// Delete the following
		if err := factory.Following().Delete(&following, "Deleted by User"); err != nil {
			return derp.Wrap(err, location, "Error deleting following", following)
		}

		// Close the Modal Dialog and return
		render.CloseModal(ctx, "")
		return ctx.NoContent(http.StatusOK)
	}
}

// following_common is a helper method that loads all of the standard pre-requisites for following handlers
func following_common(serverFactory *server.Factory, ctx echo.Context) (*domain.Factory, model.Following, primitive.ObjectID, string, error) {

	const location = "handler.followingLoad"

	// Get the factory for this domain
	factory, err := serverFactory.ByContext(ctx)

	if err != nil {
		return nil, model.Following{}, primitive.NilObjectID, "", derp.Wrap(err, location, "Error getting server factory")
	}

	// Validate the user's session
	sterankoContext := ctx.(*steranko.Context)
	authorization := getAuthorization(sterankoContext)

	// Requre that users are signed in to use this modal
	if !authorization.IsAuthenticated() {
		return nil, model.Following{}, primitive.NilObjectID, "", derp.NewUnauthorizedError(location, "User is not authenticated", nil)
	}

	// Create/Load the following
	followingService := factory.Following()
	following := model.NewFollowing()
	followingID := ctx.Param("following")
	userID := authorization.UserID

	if err := followingService.LoadByToken(userID, followingID, &following); err != nil {
		return nil, model.Following{}, primitive.NilObjectID, "", derp.Wrap(err, location, "Error loading following", followingID)
	}

	return factory, following, userID, followingID, nil
}

// following_getForm returns a form for adding/editing following
func following_getForm(folders []form.LookupCode) form.Form {

	return form.Form{
		Schema: schema.New(model.FollowingSchema()),
		Element: form.Element{
			Type: "layout-tabs",
			Children: []form.Element{
				{
					Type:  "layout-vertical",
					Label: "Settings",
					Children: []form.Element{
						{
							Type:        "text",
							Label:       "Website URL",
							Path:        "url",
							Description: "Enter the URL of the website you want to subscribe to.",
						},
						{
							Type:  "select",
							Label: "Folder",
							Path:  "folderId",
							Options: maps.Map{
								"enum": folders,
							},
							Description: "Automatically add items to this folder.",
						},
						{
							Type:        "select",
							Label:       "Poll Frequency",
							Description: "How often should this site be checked for new articles?",
							Path:        "pollDuration",
							Options: maps.Map{
								"enum": []form.LookupCode{
									{Value: "1", Label: "Hourly"},
									{Value: "6", Label: "Every 6 Hours"},
									{Value: "12", Label: "Every 12 Hours"},
									{Value: "24", Label: "Once per Day"},
									{Value: "168", Label: "Once per Week"},
									{Value: "720", Label: "Once per Month"},
								},
							},
						},
						{
							Type:        "select",
							Label:       "Remove After",
							Description: "Read items will be automatically deleted after this amount of time.",
							Path:        "purgeDuration",
							Options: maps.Map{
								"enum": []form.LookupCode{
									{Value: "1", Label: "1 Day"},
									{Value: "7", Label: "1 Week"},
									{Value: "14", Label: "2 Weeks"},
									{Value: "30", Label: "1 Month"},
									{Value: "60", Label: "2 Months"},
									{Value: "90", Label: "3 Months"},
									{Value: "180", Label: "6 Months"},
									{Value: "365", Label: "1 Year"},
								},
							},
						},
					},
				},
				{
					Type:     "layout-vertical",
					Label:    "Status",
					ReadOnly: true,
					Children: []form.Element{
						{
							Type:  "text",
							Label: "Status",
							Path:  "status",
						},
						{
							Type:  "text",
							Label: "Method",
							Path:  "method",
						},
						{
							Type:  "textarea",
							Label: "Error Details",
							Path:  "statusMessage",
						},
					},
				},
			},
		},
	}
}

// following_folderOptions returns an array of form.LookupCodes that represents all of the folders
// that belong to the currently logged in user.
func following_folderOptions(folderService *service.Folder, authenticatedID primitive.ObjectID) []form.LookupCode {

	folders, _ := folderService.QueryByUserID(authenticatedID)
	result := make([]form.LookupCode, len(folders))

	for index, folder := range folders {
		result[index] = folder.LookupCode()
	}

	return result
}
