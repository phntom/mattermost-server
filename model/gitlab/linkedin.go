package oauthgitlab

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mattermost/mattermost-server/v5/einterfaces"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/shared/mlog"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	UserAuthServiceLinkedIn = "linkedin"
	LinkedInEmailUrl        = "https://api.linkedin.com/v2/emailAddress?q=members&projection=(elements*(handle~))"
)

type LinkedInProvider struct {
	config *model.Config
}

type LinkedInUser struct {
	Id             string                 `json:"id"`
	FirstName      LinkedInName           `json:"firstName"`
	LastName       LinkedInName           `json:"lastName"`
	ProfilePicture LinkedInProfilePicture `json:"profilePicture"`
	Email          string                 `json:"email"`
}

type LinkedInName struct {
	Localized       map[string]string       `json:"localized"`
	PreferredLocale LinkedInPreferredLocale `json:"preferredLocale"`
}

type LinkedInPreferredLocale struct {
	Country  string `json:"country"`
	Language string `json:"language"`
}

type LinkedInProfilePicture struct {
	DisplayImage         string                         `json:"displayImage"`
	DisplayImageElements LinkedInProfilePictureElements `json:"displayImage~"`
}

type LinkedInProfilePictureElements struct {
	Elements []LinkedInProfilePictureEntry `json:"elements"`
}

type LinkedInProfilePictureEntry struct {
	Identifiers []LinkedInProfilePictureIdentifier `json:"identifiers"`
}

type LinkedInProfilePictureIdentifier struct {
	Identifier string `json:"identifier"`
}

type LinkedInElements struct {
	Elements []LinkedInHandle `json:"elements"`
}

type LinkedInHandle struct {
	Handle LinkedInHandleEmail `json:"handle~"`
}

type LinkedInHandleEmail struct {
	EmailAddress string `json:"emailAddress"`
}

func init() {
	mlog.Info("LinkedIn provider is initialized")
	provider := &LinkedInProvider{}
	einterfaces.RegisterOauthProvider(UserAuthServiceLinkedIn, provider)
}

func userFromLinkedInUser(liu *LinkedInUser) *model.User {
	user := &model.User{}
	user.Username = getRandomUsername(liu.Email)
	locale := fmt.Sprintf("%s_%s", liu.FirstName.PreferredLocale.Language, liu.FirstName.PreferredLocale.Country)
	user.FirstName = liu.FirstName.Localized[locale]
	user.LastName = liu.LastName.Localized[locale]
	user.SetProp(SSOPreviousFirstName, user.FirstName)
	user.SetProp(SSOPreviousLastName, user.LastName)
	user.Email = liu.Email
	if len(liu.ProfilePicture.DisplayImageElements.Elements) > 0 && len(liu.ProfilePicture.DisplayImageElements.Elements[0].Identifiers) > 0 && len(liu.ProfilePicture.DisplayImageElements.Elements[0].Identifiers[0].Identifier) > 0 {
		user.SetProp(PictureURL, liu.ProfilePicture.DisplayImageElements.Elements[0].Identifiers[0].Identifier)
	}
	userId := liu.getAuthData()
	user.AuthData = &userId
	user.AuthService = UserAuthServiceLinkedIn

	return user
}

func linkedinUserFromJson(data io.Reader) (*LinkedInUser, error) {
	var liu *LinkedInUser
	err := json.NewDecoder(data).Decode(&liu)
	return liu, err
}

func (liu *LinkedInUser) getAuthData() string {
	return liu.Id
}

func (liu *LinkedInUser) ToJson() string {
	b, err := json.Marshal(liu)
	if err != nil {
		return ""
	} else {
		return string(b)
	}
}

func (liu *LinkedInUser) IsValid() error {
	if len(liu.Id) == 0 {
		return errors.New("user id is zero")
	}

	if len(liu.Email) == 0 {
		return errors.New("no email detected")
	}

	return nil
}

func (m *LinkedInProvider) GetUserFromJson(data io.Reader, tokenUser *model.User) (*model.User, error) {
	liu, err := linkedinUserFromJson(data)
	if err != nil {
		return nil, err
	}

	if tokenUser != nil && len(tokenUser.Email) > 0 {
		liu.Email = tokenUser.Email
	}

	if err = liu.IsValid(); err != nil {
		return nil, err
	}

	return userFromLinkedInUser(liu), nil
}

func (m *LinkedInProvider) GetSSOSettings(config *model.Config, service string) (*model.SSOSettings, error) {
	if service == UserAuthServiceLinkedIn {
		m.config = config
		return &config.LinkedInSettings, nil
	}
	return nil, errors.New("no match for requested service")
}

func (m *LinkedInProvider) GetUserFromIdToken(idToken string) (*model.User, error) {
	if strings.HasPrefix(idToken, "Bearer ") {
		req, requestErr := http.NewRequest("GET", LinkedInEmailUrl, strings.NewReader(""))
		if requestErr != nil {
			return nil, requestErr
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", idToken)
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		} else if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			var lie *LinkedInElements
			err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&lie)
			if err != nil {
				return nil, err
			}
			if len(lie.Elements) > 0 && len(lie.Elements[0].Handle.EmailAddress) > 0 {
				user := &model.User{}
				user.Email = lie.Elements[0].Handle.EmailAddress
				user.SetProp(BearerToken, idToken)
				return user, nil
			}
		} else {
			_ = resp.Body.Close()
		}
	}
	return nil, nil
}

func (m *LinkedInProvider) IsSameUser(dbUser, oAuthUser *model.User) bool {
	return dbUser.AuthService == oAuthUser.AuthService &&
		dbUser.AuthData == oAuthUser.AuthData
}
