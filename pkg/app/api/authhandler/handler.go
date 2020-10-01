// Copyright 2020 The PipeCD Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authhandler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/pipe-cd/pipe/pkg/config"
	"github.com/pipe-cd/pipe/pkg/crypto"
	"github.com/pipe-cd/pipe/pkg/jwt"
	"github.com/pipe-cd/pipe/pkg/model"
)

const (
	rootPath = "/"
	// loginPath is the path to login to pipecd projects.
	loginPath = "/auth/login"
	// staticLoginPath is the path to login to pipecd projects with password.
	staticLoginPath = loginPath + "/static"
	// callbackPath is the path configured in the GitHub oauth application settings.
	callbackPath = "/auth/callback"
	// logoutPath is the path for logging out from current session.
	logoutPath = "/auth/logout"

	projectFormKey  = "project"
	usernameFormKey = "username"
	passwordFormKey = "password"
	authCodeFormKey = "code"
	stateFormKey    = "state"

	stateCookieKey = "state"
	errorCookieKey = "error"

	defaultTokenTTL          = 7 * 24 * time.Hour
	defaultStateCookieMaxAge = 30 * 60
	defaultErrorCookieMaxAge = 10 * 60
	defaultTokenCookieMaxAge = 7 * 24 * 60 * 60
)

type projectGetter interface {
	GetProject(ctx context.Context, id string) (*model.Project, error)
}

// Handler handles all imcoming requests about authentication.
type Handler struct {
	signer           jwt.Signer
	decrypter        crypto.Decrypter
	callbackURL      string
	stateKey         string
	projectsInConfig map[string]config.ControlPlaneProject
	sharedSSOConfigs map[string]*model.ProjectSSOConfig
	projectGetter    projectGetter
	secureCookie     bool
	logger           *zap.Logger
}

// NewHandler returns a handler that will used for authentication.
func NewHandler(
	signer jwt.Signer,
	decrypter crypto.Decrypter,
	address string,
	stateKey string,
	projectsInConfig map[string]config.ControlPlaneProject,
	sharedSSOConfigs map[string]*model.ProjectSSOConfig,
	projectGetter projectGetter,
	secureCookie bool,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		signer:           signer,
		decrypter:        decrypter,
		callbackURL:      strings.TrimSuffix(address, "/") + callbackPath,
		stateKey:         stateKey,
		projectsInConfig: projectsInConfig,
		sharedSSOConfigs: sharedSSOConfigs,
		projectGetter:    projectGetter,
		secureCookie:     secureCookie,
		logger:           logger,
	}
}

// Register registers all handler into the specified registry.
func (h *Handler) Register(r func(string, func(http.ResponseWriter, *http.Request))) {
	r(loginPath, h.handleSSOLogin)
	r(staticLoginPath, h.handleStaticAdminLogin)
	r(callbackPath, h.handleCallback)
	r(logoutPath, h.handleLogout)
}

// handleLogout cleans current cookies and redirects to login page.
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	http.SetCookie(w, makeExpiredTokenCookie(h.secureCookie))
	http.SetCookie(w, makeExpiredStateCookie(h.secureCookie))

	http.Redirect(w, r, rootPath, http.StatusFound)
}

func (h *Handler) findSSOConfig(p *model.Project) (*model.ProjectSSOConfig, error) {
	if p.SharedSsoName == "" {
		if p.Sso == nil {
			return nil, fmt.Errorf("missing SSO configuration in project data")
		}
		return p.Sso, nil
	}

	sso, ok := h.sharedSSOConfigs[p.SharedSsoName]
	if ok {
		return sso, nil
	}
	return nil, fmt.Errorf("not found shared sso configuration %s", p.SharedSsoName)
}

// handleError redirects to the root path and saves the error message to the cookie.
// Web will use that cookie data to handle auth error.
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, responseMessage string, err error) {
	if err != nil {
		h.logger.Error(fmt.Sprintf("auth-handler: %s", responseMessage), zap.Error(err))
	} else {
		h.logger.Info(fmt.Sprintf("auth-handler: %s", responseMessage))
	}

	http.SetCookie(w, makeErrorCookie(responseMessage, h.secureCookie))
	http.Redirect(w, r, rootPath, http.StatusSeeOther)
}

func makeTokenCookie(value string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     jwt.SignedTokenKey,
		Value:    value,
		MaxAge:   defaultTokenCookieMaxAge,
		Path:     rootPath,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func makeExpiredTokenCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     jwt.SignedTokenKey,
		Value:    "",
		MaxAge:   -1,
		Path:     rootPath,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func makeStateCookie(value string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     stateCookieKey,
		Value:    value,
		MaxAge:   defaultStateCookieMaxAge,
		Path:     rootPath,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func makeExpiredStateCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     stateCookieKey,
		Value:    "",
		MaxAge:   -1,
		Path:     rootPath,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func makeErrorCookie(value string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     errorCookieKey,
		Value:    value,
		MaxAge:   defaultErrorCookieMaxAge,
		Path:     rootPath,
		Secure:   secure,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	}
}
