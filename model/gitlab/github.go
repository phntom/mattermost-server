// Copyright (c) 2015 Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package oauthgitlab

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mattermost/mattermost-server/v5/shared/mlog"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-server/v5/einterfaces"
	"github.com/mattermost/mattermost-server/v5/model"
)

const (
	UserAuthServiceGitHub = "github"
	GitHubEmailUrl        = "https://api.github.com/user/emails?per_page=100"
	GitHubTwitterUsername = "twitter_username"
	GitHubCompany         = "company"
	GitHubHireable        = "hireable"
)

type GitHubProvider struct {
}

type GitHubUser struct {
	Id                      int64  `json:"id"`
	AvatarURL               string `json:"avatar_url"`
	Login                   string `json:"login"`
	Type                    string `json:"type"`
	Name                    string `json:"name"`
	Email                   string `json:"email"`
	TwitterUsername         string `json:"twitter_username"`
	Hireable                bool   `json:"hireable"`
	Company                 string `json:"company"`
	CreatedAt               string `json:"created_at"`
	UpdatedAt               string `json:"updated_at"`
	TwoFactorAuthentication bool   `json:"two_factor_authentication"`
}

type GitHubEmail struct {
	Email      string `json:"email"`
	Primary    bool   `json:"primary"`
	Verified   bool   `json:"verified"`
	Visibility string `json:"visibility"`
}

func init() {
	mlog.Info("GitHub provider is initialized")
	provider := &GitHubProvider{}
	einterfaces.RegisterOauthProvider(UserAuthServiceGitHub, provider)
}

func userFromGitHubUser(gou *GitHubUser, email string) *model.User {
	user := &model.User{}

	user.Username = model.CleanUsername(gou.Login)
	splitName := strings.Split(gou.Name, " ")
	if len(splitName) == 2 {
		user.FirstName = splitName[0]
		user.LastName = splitName[1]
	} else if len(splitName) >= 2 {
		user.FirstName = splitName[0]
		user.LastName = strings.Join(splitName[1:], " ")
	} else {
		user.FirstName = gou.Name
	}
	user.SetProp(SSOPreviousFirstName, user.FirstName)
	user.SetProp(SSOPreviousLastName, user.LastName)
	user.Email = strings.ToLower(email)
	user.MfaActive = gou.TwoFactorAuthentication
	if len(gou.AvatarURL) > 0 {
		user.SetProp(PictureURL, gou.AvatarURL)
	}
	if len(gou.TwitterUsername) > 0 {
		user.SetProp(GitHubTwitterUsername, gou.TwitterUsername)
	}
	if len(gou.Company) > 0 {
		user.SetProp(GitHubCompany, gou.Company)
	}
	if gou.Hireable {
		user.SetProp(GitHubHireable, "yes")
	}
	userId := gou.getAuthData()
	user.AuthData = &userId
	user.AuthService = UserAuthServiceGitHub

	return user
}

func githubUserFromJson(data io.Reader) (*GitHubUser, error) {
	decoder := json.NewDecoder(data)
	var gou GitHubUser
	err := decoder.Decode(&gou)
	if err != nil {
		return nil, err
	}
	return &gou, nil
}

func (gou *GitHubUser) ToJson() string {
	b, err := json.Marshal(gou)
	if err != nil {
		return ""
	}
	return string(b)
}

func (gou *GitHubUser) IsValid() error {
	if gou.Id == 0 {
		return errors.New("user sub can't be empty")
	}

	if gou.Type != "User" {
		return errors.New("user type is not User")
	}

	return nil
}

func (gou *GitHubUser) getAuthData() string {
	return fmt.Sprintf("%d", gou.Id)
}

func (m *GitHubProvider) GetUserFromJson(data io.Reader, tokenUser *model.User) (*model.User, error) {
	gou, err := githubUserFromJson(data)
	if err != nil {
		return nil, err
	}
	if err = gou.IsValid(); err != nil {
		return nil, err
	}

	if tokenUser == nil || len(tokenUser.Email) == 0 {
		return nil, errors.New("couldn't detect user email")
	}

	return userFromGitHubUser(gou, tokenUser.Email), nil
}

func (m *GitHubProvider) GetSSOSettings(config *model.Config, service string) (*model.SSOSettings, error) {
	return &config.GitHubSettings, nil
}

func (m *GitHubProvider) GetUserFromIdToken(idToken string) (*model.User, error) {
	if strings.HasPrefix(idToken, "Bearer ") {
		req, requestErr := http.NewRequest("GET", GitHubEmailUrl, strings.NewReader(""))
		if requestErr != nil {
			return nil, requestErr
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("Authorization", idToken)
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		} else if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			var ghe []GitHubEmail
			err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&ghe)
			if err != nil {
				return nil, err
			}
			for _, email := range ghe {
				if email.Verified {
					user := &model.User{}
					user.Email = email.Email
					user.SetProp(BearerToken, idToken)
					return user, nil
				}
			}
		} else {
			_ = resp.Body.Close()
		}
	}
	return nil, nil
}

func (m *GitHubProvider) IsSameUser(dbUser, oauthUser *model.User) bool {
	return dbUser.AuthService == oauthUser.AuthService &&
		dbUser.AuthData == oauthUser.AuthData
}
