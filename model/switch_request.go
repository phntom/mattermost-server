// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

type SwitchRequest struct {
	CurrentService string `json:"current_service"`
	NewService     string `json:"new_service"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	NewPassword    string `json:"new_password"`
	MfaCode        string `json:"mfa_code"`
	LdapLoginId    string `json:"ldap_id"`
}

func (o *SwitchRequest) EmailToOAuth() bool {
	return o.CurrentService == UserAuthServiceEmail &&
		(o.NewService == UserAuthServiceSaml ||
			o.NewService == UserAuthServiceGitlab ||
			o.NewService == ServiceGoogle ||
			o.NewService == ServiceOffice365 ||
			o.NewService == ServiceOpenid ||
			o.NewService == ServiceFacebook ||
			o.NewService == ServiceLinkedin ||
			o.NewService == ServiceGithub ||
			o.NewService == ServiceTwitter)
}

func (o *SwitchRequest) OAuthToEmail() bool {
	return (o.CurrentService == UserAuthServiceSaml ||
		o.CurrentService == UserAuthServiceGitlab ||
		o.CurrentService == ServiceGoogle ||
		o.CurrentService == ServiceOffice365 ||
		o.CurrentService == ServiceOpenid ||
		o.CurrentService == ServiceFacebook ||
		o.CurrentService == ServiceLinkedin ||
		o.CurrentService == ServiceGithub ||
		o.CurrentService == ServiceTwitter) && o.NewService == UserAuthServiceEmail
}

func (o *SwitchRequest) EmailToLdap() bool {
	return o.CurrentService == UserAuthServiceEmail && o.NewService == UserAuthServiceLdap
}

func (o *SwitchRequest) LdapToEmail() bool {
	return o.CurrentService == UserAuthServiceLdap && o.NewService == UserAuthServiceEmail
}
