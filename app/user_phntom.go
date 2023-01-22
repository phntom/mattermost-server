package app

import (
	"github.com/mattermost/mattermost-server/v6/app/request"
	"github.com/mattermost/mattermost-server/v6/einterfaces"
	"github.com/mattermost/mattermost-server/v6/model"
	oauthgitlab "github.com/mattermost/mattermost-server/v6/model/oauthproviders/gitlab"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
	"github.com/mattermost/mattermost-server/v6/store"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"strconv"
)

func (a *App) CreateOAuthUser(c *request.Context, service string, userData io.Reader, teamID string, tokenUser *model.User) (*model.User, *model.AppError) {
	if !*a.Config().TeamSettings.EnableUserCreation {
		return nil, model.NewAppError("CreateOAuthUser", "api.user.create_user.disabled.app_error", nil, "", http.StatusNotImplemented)
	}

	provider, e := a.getSSOProvider(service)
	if e != nil {
		return nil, e
	}
	user, err1 := provider.GetUserFromJSON(userData, tokenUser)
	if err1 != nil {
		return nil, model.NewAppError("CreateOAuthUser", "api.user.create_oauth_user.create.app_error", map[string]any{"Service": service}, "", http.StatusInternalServerError).Wrap(err1)
	}
	if user.AuthService == "" {
		user.AuthService = service
	}

	found := true
	count := 0
	for found {
		if found = a.ch.srv.userService.IsUsernameTaken(user.Username); found {
			user.Username = user.Username + strconv.Itoa(count)
			count++
		}
	}

	userByAuth, _ := a.ch.srv.userService.GetUserByAuth(user.AuthData, service)
	if userByAuth != nil {
		return userByAuth, nil
	}

	userByEmail, _ := a.ch.srv.userService.GetUserByEmail(user.Email)
	if userByEmail != nil {
		if userByEmail.AuthService == "" {
			return nil, model.NewAppError("CreateOAuthUser", "api.user.create_oauth_user.already_attached.app_error", map[string]any{"Service": service, "Auth": model.UserAuthServiceEmail}, "email="+user.Email, http.StatusBadRequest)
		}
		if provider.IsSameUser(userByEmail, user) {
			if _, err := a.Srv().Store().User().UpdateAuthData(userByEmail.Id, user.AuthService, user.AuthData, "", false); err != nil {
				// if the user is not updated, write a warning to the log, but don't prevent user login
				c.Logger().Warn("Error attempting to update user AuthData", mlog.Err(err))
			}
			return userByEmail, nil
		}
		return nil, model.NewAppError("CreateOAuthUser", "api.user.create_oauth_user.already_attached.app_error", map[string]any{"Service": service, "Auth": userByEmail.AuthService}, "email="+user.Email+" authData="+*user.AuthData, http.StatusBadRequest)
	}

	user.EmailVerified = true

	ruser, err := a.CreateUser(c, user)
	if err != nil {
		return nil, err
	}

	appError, _ := a.updateUserPicture(user)
	if appError != nil {
		return nil, model.NewAppError("CreateOAuthUser", "api.user.create_oauth_user.create.app_error", map[string]interface{}{"Service": service}, appError.Error(), http.StatusInternalServerError)
	}

	if teamID != "" {
		err = a.AddUserToTeamByTeamId(c, teamID, user)
		if err != nil {
			return nil, err
		}

		err = a.AddDirectChannels(c, teamID, user)
		if err != nil {
			c.Logger().Warn("Failed to add direct channels", mlog.Err(err))
		}
	}

	return ruser, nil
}

func (a *App) UpdateOAuthUserAttrs(userData io.Reader, user *model.User, provider einterfaces.OAuthProvider, service string, tokenUser *model.User) *model.AppError {
	oauthUser, err1 := provider.GetUserFromJSON(userData, tokenUser)
	if err1 != nil {
		return model.NewAppError("UpdateOAuthUserAttrs", "api.user.update_oauth_user_attrs.get_user.app_error", map[string]any{"Service": service}, "", http.StatusBadRequest).Wrap(err1)
	}

	userAttrsChanged := false

	//if oauthUser.Username != user.Username {
	//	if existingUser, _ := a.GetUserByUsername(oauthUser.Username); existingUser == nil {
	//		user.Username = oauthUser.Username
	//		userAttrsChanged = true
	//	}
	//}

	if oauthUser.GetFullName() != user.GetFullName() {
		if prevFirstName, ok := user.GetProp(oauthgitlab.SSOPreviousFirstName); !ok || prevFirstName != oauthUser.FirstName {
			user.FirstName = oauthUser.FirstName
			user.SetProp(oauthgitlab.SSOPreviousFirstName, oauthUser.FirstName)
			userAttrsChanged = true
		}
		if prevLastName, ok := user.GetProp(oauthgitlab.SSOPreviousLastName); !ok || prevLastName != oauthUser.LastName {
			user.LastName = oauthUser.LastName
			user.SetProp(oauthgitlab.SSOPreviousLastName, oauthUser.LastName)

		}
	}

	if oauthUser.Email != user.Email {
		if existingUser, _ := a.GetUserByEmail(oauthUser.Email); existingUser == nil {
			user.Email = oauthUser.Email
			userAttrsChanged = true
		}
	}

	oauthPicUrl, okOauthPicUrl := oauthUser.GetProp(oauthgitlab.PictureURL)
	userPicUrl, okUserPicUrl := user.GetProp(oauthgitlab.PictureURL)
	if okOauthPicUrl && okUserPicUrl && oauthPicUrl != userPicUrl {
		err, pictureChanged := a.updateUserPicture(user)
		if err != nil {
			return model.NewAppError("UpdateOAuthUserAttrs", "app.user.update.finding.app_error", nil, err.Error(), http.StatusInternalServerError)
		}
		userAttrsChanged = userAttrsChanged || pictureChanged
	}

	if user.DeleteAt > 0 {
		// Make sure they are not disabled
		user.DeleteAt = 0
		userAttrsChanged = true
	}

	if userAttrsChanged {
		users, err := a.Srv().Store().User().Update(user, true)
		if err != nil {
			var appErr *model.AppError
			var invErr *store.ErrInvalidInput
			switch {
			case errors.As(err, &appErr):
				return appErr
			case errors.As(err, &invErr):
				return model.NewAppError("UpdateOAuthUserAttrs", "app.user.update.find.app_error", nil, "", http.StatusBadRequest).Wrap(err)
			default:
				return model.NewAppError("UpdateOAuthUserAttrs", "app.user.update.finding.app_error", nil, "", http.StatusInternalServerError).Wrap(err)
			}
		}

		user = users.New
		a.InvalidateCacheForUser(user.Id)
	}

	return nil
}

func (a *App) CheckProviderAttributes(user *model.User, patch *model.UserPatch) string {
	//tryingToChange := func(userValue *string, patchValue *string) bool {
	//	return patchValue != nil && *patchValue != *userValue
	//}

	//// If any login provider is used, then the username may not be changed
	//if user.AuthService != "" && tryingToChange(&user.Username, patch.Username) {
	//	return "username"
	//}

	LdapSettings := &a.Config().LdapSettings
	SamlSettings := &a.Config().SamlSettings

	conflictField := ""
	if a.Ldap() != nil &&
		(user.IsLDAPUser() || (user.IsSAMLUser() && *SamlSettings.EnableSyncWithLdap)) {
		conflictField = a.Ldap().CheckProviderAttributes(LdapSettings, user, patch)
	} else if a.Saml() != nil && user.IsSAMLUser() {
		conflictField = a.Saml().CheckProviderAttributes(SamlSettings, user, patch)
		//} else if user.IsOAuthUser() {
		//	if tryingToChange(&user.FirstName, patch.FirstName) || tryingToChange(&user.LastName, patch.LastName) {
		//		conflictField = "full name"
		//	}
	}

	return conflictField
}

func (a *App) updateUserPicture(user *model.User) (error, bool) {
	picUrl, ok := user.GetProp(oauthgitlab.PictureURL)
	if !ok {
		return nil, false
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", picUrl, nil)
	if err != nil {
		return err, false
	}
	resp, err := client.Do(req)
	if err != nil {
		return err, false
	}
	if resp.StatusCode == 200 {
		defer resp.Body.Close()

		path := "users/" + user.Id + "/profile.png"
		if _, err := a.WriteFile(resp.Body, path); err != nil {
			return err, false
		}

		if err := a.Srv().Store().User().UpdateLastPictureUpdate(user.Id); err != nil {
			return err, false
		}
	}
	return nil, true
}
