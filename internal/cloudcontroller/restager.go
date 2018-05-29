package cloudcontroller

import (
	"fmt"
	"net/http"
)

type Restager struct {
	c       AuthCurler
	log     Logger
	appGUID string
}

func NewRestager(appGUID string, c AuthCurler, log Logger) *Restager {
	return &Restager{
		c:       c,
		log:     log,
		appGUID: appGUID,
	}
}

func (r *Restager) SaveAndRestage(refreshToken string) {
	r.saveRefreshToken(refreshToken)
	r.restageApp()
}

func (r *Restager) saveRefreshToken(refreshToken string) {
	url := fmt.Sprintf("/v3/apps/%s/environment_variables", r.appGUID)
	body := fmt.Sprintf(`{"var":{"REFRESH_TOKEN": %q}}`, refreshToken)
	_, err := r.c.Curl(url, http.MethodPatch, body)
	if err != nil {
		r.log.Fatalf("Failed to updated REFRESH_TOKEN with cloud controller: %s", err)
	}
}

// Restage to enable the app to start with the new refresh token. This
// ensures that if the app crashes or gets restarted, it will have proper
// state.
func (r *Restager) restageApp() {
	url := fmt.Sprintf("/v2/apps/%s/restage", r.appGUID)
	_, err := r.c.Curl(url, http.MethodPost, "")
	if err != nil {
		r.log.Fatalf("Failed to restage app: %s", err)
	}
}
