package model

import (
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/slice"
)

// https://docs.joinmastodon.org/methods/oauth/#token
// POST /oauth/token
// Returns: Token
// Obtain an access token, to be used during API calls that are not public
type OAuthUserTokenRequest struct {
	GrantType    string `form:"grant_type"`
	Code         string `form:"code"`
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
	RedirectURI  string `form:"redirect_uri"`
	Scope        string `form:"scope"`
}

func NewOAuthUserTokenRequest() OAuthUserTokenRequest {
	return OAuthUserTokenRequest{}
}

func (o OAuthUserTokenRequest) Scopes() []string {
	if o.Scope == "" {
		return []string{"read"}
	}

	scope := strings.ReplaceAll(o.Scope, ",", " ")
	return strings.Split(scope, " ")
}

// Validate confirms that a request is valid based on the settings in the OAuthApplication.
// This method MAY update the request if certain values are missing.
func (req *OAuthUserTokenRequest) Validate(app OAuthApplication) error {

	const location = "model.OAuthUserTokenRequest.Validate"

	if len(app.RedirectURIs) == 0 {
		return derp.NewInternalError(location, "Application must have at least one redirect_uri")
	}

	// RULE: If missing, use default value for RedirectURI
	if req.RedirectURI == "" {
		req.RedirectURI = app.RedirectURIs[0]
	}

	// RULE: Verify that redirect URI is valid
	if !slice.Contains(app.RedirectURIs, req.RedirectURI) {
		return derp.NewBadRequestError(location, "Invalid redirect_uri", app, req)
	}

	// RULE: If missing, use default value for Scope
	if req.Scope == "" {
		req.Scope = strings.Join(app.Scopes, " ")
	}

	// RULE: Verify that scope is valid
	for _, scope := range req.Scopes() {
		if !slice.Contains(app.Scopes, scope) {
			return derp.NewBadRequestError(location, "Invalid scope", scope)
		}
	}

	// RULE: ResponseType must be one of the approved values.
	switch req.GrantType {
	case "code":
	case "token":
	default:
		req.GrantType = "code"
	}

	// Success
	return nil
}
