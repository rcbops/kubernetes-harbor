package models

type OIDCSettings struct {
	KeyType string
	Key     interface{}
}

type OAuthSettings struct {
	ClientID     string
	ClientSecret string
	Certificate  string
	AuthURL      string
	TokenURL     string
	OIDC         OIDCSettings
}
