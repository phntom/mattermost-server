package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mattermost/mattermost-server/v6/model"
	"net/http"
	"strings"
)

func (a *App) rawSendToPushProxy(msg *model.PushNotification) (model.PushResponse, error) {
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode to JSON: %w", err)
	}

	serverPrefix := *a.Config().EmailSettings.PushNotificationServer
	if strings.Contains(msg.Platform, "custom") {
		serverPrefix = *a.Config().EmailSettings.PushNotificationServerCustom
	}
	url := strings.TrimRight(serverPrefix, "/") + model.APIURLSuffixV1 + "/send_push"
	request, err := http.NewRequest("POST", url, bytes.NewReader(msgJSON))
	if err != nil {
		return nil, err
	}

	resp, err := a.Srv().pushNotificationClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pushResponse model.PushResponse
	if err := json.NewDecoder(resp.Body).Decode(&pushResponse); err != nil {
		return nil, fmt.Errorf("failed to decode from JSON: %w", err)
	}

	return pushResponse, nil
}
