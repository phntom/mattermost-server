package model

const (
	ServiceFacebook = "facebook"
	ServiceLinkedin = "linkedin"
	ServiceGithub   = "github"
	ServiceTwitter  = "twitter"
)

type EmailSettings struct {
	EmailSettingsPhntom
	PushNotificationServerCustom *string `access:"environment_push_notification_server"` // telemetry: none
}

type Config struct {
	ConfigPhntom
	LinkedInSettings SSOSettings
	FacebookSettings SSOSettings
	GitHubSettings   SSOSettings
	TwitterSettings  SSOSettings
}

func (o *Config) GetSSOService(service string) *SSOSettings {
	switch service {
	case ServiceFacebook:
		return &o.FacebookSettings
	case ServiceLinkedin:
		return &o.LinkedInSettings
	case ServiceGithub:
		return &o.GitHubSettings
	case ServiceTwitter:
		return &o.TwitterSettings
	}

	return o.GetSSOServicePhntom(service)
}

func (o *Config) SetDefaults() {
	o.SetDefaultsPhntom()
	o.FacebookSettings.setDefaults("", "", "", "", "")
	o.LinkedInSettings.setDefaults("", "", "", "", "")
	o.GitHubSettings.setDefaults("", "", "", "", "")
	o.TwitterSettings.setDefaults("", "", "", "", "")
}
