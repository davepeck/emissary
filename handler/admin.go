package handler

import (
	"github.com/EmissarySocial/emissary/builder"
	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/server"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetAdmin handles GET requests
func GetAdmin(factoryManager *server.Factory) echo.HandlerFunc {
	return buildAdmin(factoryManager, builder.ActionMethodGet)
}

// PostAdmin handles POST/DELETE requests
func PostAdmin(factoryManager *server.Factory) echo.HandlerFunc {
	return buildAdmin(factoryManager, builder.ActionMethodPost)
}

func buildAdmin(factoryManager *server.Factory, actionMethod builder.ActionMethod) echo.HandlerFunc {

	const location = "handler.adminBuilder"

	return func(ctx echo.Context) error {

		// Authenticate the page request
		sterankoContext := ctx.(*steranko.Context)

		if !isOwner(sterankoContext.Authorization()) {
			return derp.NewForbiddenError(location, "Unauthorized")
		}

		// Try to get the factory from the Context
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized Domain")
		}

		// Parse admin parameters
		templateID, actionID, objectID := buildAdmin_ParsePath(ctx)

		// Try to load the Template
		templateService := factory.Template()
		template, err := templateService.LoadAdmin(templateID)

		if err != nil {
			return err
		}

		// Locate and populate the builder
		builder, err := buildAdmin_GetBuilder(factory, sterankoContext, template, actionID, objectID)

		if err != nil {
			return derp.Wrap(err, location, "Error generating builder")
		}

		// Success!!
		return buildHTML(factory, sterankoContext, builder, actionMethod)
	}
}

func buildAdmin_ParsePath(ctx echo.Context) (string, string, primitive.ObjectID) {

	// First parameter is always the templateID
	templateID := first.String(ctx.Param("param1"), "domain")

	// If the second parameter is an ObjectID, then we parse object/action
	if objectID, err := primitive.ObjectIDFromHex(ctx.Param("param2")); err == nil {
		actionID := first.String(ctx.Param("param3"), "view")

		return templateID, actionID, objectID
	}

	// Otherwise, we just parse action
	actionID := first.String(ctx.Param("param2"), "index")
	return templateID, actionID, primitive.NilObjectID
}

func buildAdmin_GetBuilder(factory *domain.Factory, ctx *steranko.Context, template model.Template, actionID string, objectID primitive.ObjectID) (builder.Builder, error) {

	const location = "handler.buildAdmin_GetBuilder"

	// Create the correct builder for this controller
	switch template.Model {

	case "rule":

		ruleService := factory.Rule()
		rule := model.NewRule()

		if !objectID.IsZero() {
			authorization := getAuthorization(ctx)
			if err := ruleService.LoadByID(authorization.UserID, objectID, &rule); err != nil {
				return nil, derp.Wrap(err, location, "Error loading Rule", objectID)
			}
		}

		return builder.NewRule(factory, ctx.Request(), ctx.Response(), &rule, template, actionID)

	case "domain":
		return builder.NewDomain(factory, ctx.Request(), ctx.Response(), template, actionID)

	case "group":
		group := model.NewGroup()

		if !objectID.IsZero() {
			service := factory.Group()
			if err := service.LoadByID(objectID, &group); err != nil {
				return nil, derp.Wrap(err, location, "Error loading Group", objectID)
			}
		}

		return builder.NewGroup(factory, ctx.Request(), ctx.Response(), template, &group, actionID)

	case "stream":
		stream := model.NewStream()

		if !objectID.IsZero() {
			service := factory.Stream()
			if err := service.LoadByID(objectID, &stream); err != nil {
				return nil, derp.Wrap(err, location, "Error loading Navigation stream", objectID)
			}
		}

		return builder.NewNavigation(factory, ctx.Request(), ctx.Response(), template, &stream, actionID)

	case "user":
		user := model.NewUser()

		if !objectID.IsZero() {
			service := factory.User()
			if err := service.LoadByID(objectID, &user); err != nil {
				return nil, derp.Wrap(err, location, "Error loading User", objectID)
			}
		}

		return builder.NewUser(factory, ctx.Request(), ctx.Response(), template, &user, actionID)

	default:
		return nil, derp.NewNotFoundError(location, "Template MODEL must be one of: 'rule', 'domain', 'group', 'stream', or 'user'", template.Model)
	}
}
