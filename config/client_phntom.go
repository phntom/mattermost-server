package config

import (
	"github.com/mattermost/mattermost-server/v6/model"
	"strconv"
)

func GenerateLimitedClientConfig(c *model.Config, telemetryID string, license *model.License) map[string]string {
	props := GenerateLimitedClientConfig1(c, telemetryID, license)
	props["EnableSignUpWithGoogle"] = strconv.FormatBool(*c.GoogleSettings.Enable)
	props["EnableSignUpWithFacebook"] = strconv.FormatBool(*c.FacebookSettings.Enable)
	props["EnableSignUpWithLinkedIn"] = strconv.FormatBool(*c.LinkedInSettings.Enable)
	props["EnableSignUpWithGitHub"] = strconv.FormatBool(*c.GitHubSettings.Enable)
	props["EnableSignUpWithTwitter"] = strconv.FormatBool(*c.TwitterSettings.Enable)
	return props
}
