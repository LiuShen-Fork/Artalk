package generic

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/markbates/goth"
)

// Session stores OAuth authorization and token data between the login request
// and its callback.
type Session struct {
	AuthURL      string    `json:"auth_url"`
	AccessToken  string    `json:"access_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
}

var _ goth.Session = (*Session)(nil)

func (s Session) GetAuthURL() (string, error) {
	if s.AuthURL == "" {
		return "", errors.New(goth.NoAuthUrlErrorMessage)
	}
	return s.AuthURL, nil
}

func (s Session) Marshal() string {
	data, _ := json.Marshal(s)
	return string(data)
}

func (s Session) String() string {
	return s.Marshal()
}

func (s *Session) Authorize(provider goth.Provider, params goth.Params) (string, error) {
	p, ok := provider.(*Provider)
	if !ok {
		return "", errors.New("invalid generic OAuth provider")
	}
	code := strings.TrimSpace(params.Get("code"))
	if code == "" {
		return "", errors.New("authorization code is missing")
	}

	token, err := p.config.Exchange(goth.ContextForClient(p.Client()), code)
	if err != nil {
		return "", err
	}
	if !token.Valid() {
		return "", errors.New("invalid token received from provider")
	}

	s.AccessToken = token.AccessToken
	s.TokenType = token.TokenType
	s.RefreshToken = token.RefreshToken
	s.ExpiresAt = token.Expiry
	if idToken, ok := token.Extra("id_token").(string); ok {
		s.IDToken = idToken
	}
	return token.AccessToken, nil
}
