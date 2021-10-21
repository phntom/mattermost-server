package oauthgitlab

import (
	"encoding/json"
	"errors"
	"github.com/mattermost/mattermost-server/v6/einterfaces"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
	"io"
)

const (
	UserAuthServiceFacebook = "facebook"
)

type FacebookProvider struct {
	config *model.Config
}

type FacebookUser struct {
	Id        string          `json:"id"`
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	FirstName string          `json:"first_name"`
	LastName  string          `json:"last_name"`
	Picture   FacebookPicture `json:"picture"`
}

type FacebookPicture struct {
	Data FacebookPictureData `json:"data"`
}

type FacebookPictureData struct {
	Url string `json:"url"`
}

func init() {
	mlog.Info("Facebook provider is initialized")
	provider := &FacebookProvider{}
	einterfaces.RegisterOAuthProvider(UserAuthServiceFacebook, provider)
}

func userFromFacebookUser(fbu *FacebookUser) *model.User {
	user := &model.User{}
	user.Username = getRandomUsername(fbu.Email)
	user.FirstName = fbu.FirstName
	user.LastName = fbu.LastName
	user.SetProp(SSOPreviousFirstName, fbu.FirstName)
	user.SetProp(SSOPreviousLastName, fbu.LastName)
	user.Email = fbu.Email
	if len(fbu.Picture.Data.Url) > 0 {
		user.SetProp(PictureURL, fbu.Picture.Data.Url)
	}
	userId := fbu.getAuthData()
	user.AuthData = &userId
	user.AuthService = UserAuthServiceFacebook

	return user
}

func facebookUserFromJson(data io.Reader) (*FacebookUser, error) {
	var fbu *FacebookUser
	err := json.NewDecoder(data).Decode(&fbu)
	return fbu, err
}

func (fbu *FacebookUser) getAuthData() string {
	return fbu.Id
}

func (fbu *FacebookUser) ToJson() string {
	b, err := json.Marshal(fbu)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func (fbu *FacebookUser) IsValid() error {
	if len(fbu.Id) == 0 {
		return errors.New("user id is zero")
	}

	if len(fbu.Email) == 0 {
		return errors.New("email not found")
	}

	return nil
}

func (m *FacebookProvider) GetUserFromJSON(data io.Reader, tokenUser *model.User) (*model.User, error) {
	fbu, err := facebookUserFromJson(data)
	if err != nil {
		return nil, err
	}

	if err = fbu.IsValid(); err != nil {
		return nil, err
	}

	return userFromFacebookUser(fbu), nil
}

func (m *FacebookProvider) GetSSOSettings(config *model.Config, service string) (*model.SSOSettings, error) {
	if service == UserAuthServiceFacebook {
		m.config = config
		return &config.FacebookSettings, nil
	}
	return nil, errors.New("no match for requested service")
}

func (m *FacebookProvider) GetUserFromIdToken(idToken string) (*model.User, error) {
	return nil, nil
}

func (m *FacebookProvider) IsSameUser(dbUser, oAuthUser *model.User) bool {
	return dbUser.AuthService == oAuthUser.AuthService &&
		dbUser.AuthData == oAuthUser.AuthData
}
