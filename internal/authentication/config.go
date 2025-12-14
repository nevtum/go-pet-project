package authentication

import (
	"os"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	IssuerURL    string
	JWKSURL      string
}

func LoadConfig() *Config {
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	redirectURL := os.Getenv("REDIRECT_URL")
	issuerURL := os.Getenv("ISSUER_URL")
	jwksURL := os.Getenv("JWKS_URL")

	return &Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		IssuerURL:    issuerURL,
		JWKSURL:      jwksURL,
	}
}
