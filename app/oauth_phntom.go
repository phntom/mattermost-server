package app

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
	"github.com/mattermost/mattermost-server/v6/utils"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (a *App) AuthorizeOAuthUser(w http.ResponseWriter, r *http.Request, service, code, state, redirectURI string) (io.ReadCloser, string, map[string]string, *model.User, *model.AppError) {
	provider, e := a.getSSOProvider(service)
	if e != nil {
		return nil, "", nil, nil, e
	}

	sso, e2 := provider.GetSSOSettings(a.Config(), service)
	if e2 != nil {
		return nil, "", nil, nil, model.NewAppError("AuthorizeOAuthUser.GetSSOSettings", "api.user.get_authorization_code.endpoint.app_error", nil, "", http.StatusNotImplemented).Wrap(e2)
	}

	b, strErr := b64.StdEncoding.DecodeString(state)
	if strErr != nil {
		return nil, "", nil, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.invalid_state.app_error", nil, "", http.StatusBadRequest).Wrap(strErr)
	}

	stateStr := string(b)
	stateProps := model.MapFromJSON(strings.NewReader(stateStr))

	expectedToken, appErr := a.GetOAuthStateToken(stateProps["token"])
	if appErr != nil {
		return nil, "", stateProps, nil, appErr
	}

	stateEmail := stateProps["email"]
	stateAction := stateProps["action"]
	if stateAction == model.OAuthActionEmailToSSO && stateEmail == "" {
		return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.invalid_state.app_error", nil, "", http.StatusBadRequest)
	}

	cookie, cookieErr := r.Cookie(CookieOAuth)
	if cookieErr != nil {
		return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.invalid_state.app_error", nil, "", http.StatusBadRequest)
	}

	expectedTokenExtra := generateOAuthStateTokenExtra(stateEmail, stateAction, cookie.Value)
	if expectedTokenExtra != expectedToken.Extra {
		return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.invalid_state.app_error", nil, "", http.StatusBadRequest)
	}

	appErr = a.DeleteToken(expectedToken)
	if appErr != nil {
		mlog.Warn("error deleting token", mlog.Err(appErr))
	}

	subpath, _ := utils.GetSubpathFromConfig(a.Config())

	httpCookie := &http.Cookie{
		Name:     CookieOAuth,
		Value:    "",
		Path:     subpath,
		MaxAge:   -1,
		HttpOnly: true,
	}

	http.SetCookie(w, httpCookie)

	teamID := stateProps["team_id"]

	p := url.Values{}
	p.Set("client_id", *sso.Id)
	p.Set("client_secret", *sso.Secret)
	p.Set("code", code)
	p.Set("grant_type", model.AccessTokenGrantType)
	p.Set("redirect_uri", redirectURI)

	req, requestErr := http.NewRequest("POST", *sso.TokenEndpoint, strings.NewReader(p.Encode()))
	if requestErr != nil {
		return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.token_failed.app_error", nil, "", http.StatusInternalServerError).Wrap(requestErr)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := a.HTTPService().MakeClient(true).Do(req)
	if err != nil {
		return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.token_failed.app_error", nil, "", http.StatusInternalServerError).Wrap(err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	tee := io.TeeReader(resp.Body, &buf)
	var ar *model.AccessResponse
	err = json.NewDecoder(tee).Decode(&ar)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.bad_response.app_error", nil, fmt.Sprintf("response_body=%s, status_code=%d, error=%v", buf.String(), resp.StatusCode, err), http.StatusInternalServerError).Wrap(err)
	}

	if service != "linkedin" && strings.ToLower(ar.TokenType) != model.AccessTokenType {
		return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.bad_token.app_error", nil, "token_type="+ar.TokenType+", response_body="+buf.String(), http.StatusInternalServerError)
	}

	if ar.AccessToken == "" {
		return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.missing.app_error", nil, "response_body="+buf.String(), http.StatusInternalServerError)
	}

	p = url.Values{}
	p.Set("access_token", ar.AccessToken)

	var userFromToken *model.User
	userFromToken, err = provider.GetUserFromIdToken("Bearer " + ar.AccessToken)
	if err != nil {
		return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.token_failed.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	if userFromToken != nil && ar.IdToken != "" {
		userFromToken, err = provider.GetUserFromIdToken(ar.IdToken)
		if err != nil {
			return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.token_failed.app_error", nil, "", http.StatusInternalServerError).Wrap(err)
		}
	}

	req, requestErr = http.NewRequest("GET", *sso.UserAPIEndpoint, strings.NewReader(""))
	if requestErr != nil {
		return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.service.app_error", map[string]any{"Service": service}, "", http.StatusInternalServerError).Wrap(requestErr)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+ar.AccessToken)

	resp, err = a.HTTPService().MakeClient(true).Do(req)
	if err != nil {
		return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.service.app_error", map[string]any{"Service": service}, "", http.StatusInternalServerError).Wrap(err)
	} else if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()

		// Ignore the error below because the resulting string will just be the empty string if bodyBytes is nil
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)

		mlog.Error("Error getting OAuth user", mlog.Int("response", resp.StatusCode), mlog.String("body_string", bodyString))

		if service == model.ServiceGitlab && resp.StatusCode == http.StatusForbidden && strings.Contains(bodyString, "Terms of Service") {
			// Return a nicer error when the user hasn't accepted GitLab's terms of service
			return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "oauth.gitlab.tos.error", nil, "", http.StatusBadRequest)
		}

		return nil, "", stateProps, nil, model.NewAppError("AuthorizeOAuthUser", "api.user.authorize_oauth_user.response.app_error", nil, "response_body="+bodyString, http.StatusInternalServerError)
	}

	// Note that resp.Body is not closed here, so it must be closed by the caller
	return resp.Body, teamID, stateProps, userFromToken, nil
}
