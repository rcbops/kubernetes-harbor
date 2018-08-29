package models

type OAuthSigningKey struct {
	Type string
	Data interface{}
}

type OAuthSettings struct {
	ClientID     string
	ClientSecret string
	Certificate  string
	AuthURL      string
	TokenURL     string
	SigningKey   OAuthSigningKey
}
