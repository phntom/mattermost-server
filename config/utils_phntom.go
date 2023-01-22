package config

import "github.com/mattermost/mattermost-server/v6/model"

func desanitize(actual, target *model.Config) {
	if target.FacebookSettings.Secret != nil && *target.FacebookSettings.Secret == model.FakeSetting {
		target.FacebookSettings.Secret = actual.FacebookSettings.Secret
	}

	if target.LinkedInSettings.Secret != nil && *target.LinkedInSettings.Secret == model.FakeSetting {
		target.LinkedInSettings.Secret = actual.LinkedInSettings.Secret
	}

	if target.GitHubSettings.Secret != nil && *target.GitHubSettings.Secret == model.FakeSetting {
		target.GitHubSettings.Secret = actual.GitHubSettings.Secret
	}

	if target.TwitterSettings.Secret != nil && *target.TwitterSettings.Secret == model.FakeSetting {
		target.TwitterSettings.Secret = actual.TwitterSettings.Secret
	}

	desanitizePhntom(actual, target)
}
