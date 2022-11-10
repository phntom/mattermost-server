package oauthgitlab

import "github.com/mattermost/mattermost-server/v6/model"

func userFromGitLabUser(glu *GitLabUser) *model.User {
	user := userFromGitLabUser(glu)
	user.SetProp(SSOPreviousFirstName, user.FirstName)
	user.SetProp(SSOPreviousLastName, user.LastName)
	return user
}
