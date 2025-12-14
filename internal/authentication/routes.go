package authentication

import (
	"bytes"
	"context"
	"es/internal/util"
	"fmt"
	"html/template"
	"net/http"

	"github.com/coreos/go-oidc"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/oauth2"
)

type RouteHandler struct {
	cfg      *Config
	oauthCfg oauth2.Config
}

func NewRouteHandler(cfg *Config) *RouteHandler {
	provider := util.Must(oidc.NewProvider(context.Background(), cfg.IssuerURL))

	return &RouteHandler{
		cfg: cfg,
		oauthCfg: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"phone", "openid", "email"},
			Endpoint:     provider.Endpoint(),
		},
	}
}

func (h *RouteHandler) Login(c *fiber.Ctx) error {
	state := "state" // Replace with a secure random string in production
	url := h.oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return c.Redirect(url, http.StatusFound)
}

func (h *RouteHandler) Logout(c *fiber.Ctx) error {
	// Implement logout logic here
	tmpl := `
    <html>
        <body>
            <h1>Sucessfully logged out</h1>
            <a href="/login">Login</a>
        </body>
    </html>`

	// Set content type and send the rendered HTML
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(tmpl)
}

func (h *RouteHandler) Callback(c *fiber.Ctx) error {
	code := c.Query("code")

	// Exchange the authorization code for a token
	rawToken, err := h.oauthCfg.Exchange(c.Context(), code)
	if err != nil {
		return fmt.Errorf("failed to exchange token: %s", err.Error())
	}
	tokenString := rawToken.AccessToken

	// Parse the token (do signature verification for your use case in production)
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("failed to parse token: %s", err.Error())
	}

	// Check if the token is valid and extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.Status(http.StatusBadRequest).SendString("invalid claims")
	}

	type claimsPage struct {
		AccessToken string
		Claims      jwt.MapClaims
	}
	// Prepare data for rendering the template
	pageData := claimsPage{
		AccessToken: tokenString,
		Claims:      claims,
	}

	tmpl := `
    <html>
        <body>
            <h1>User Information</h1>
            <h1>JWT Claims</h1>
            <p><strong>Access Token:</strong> {{.AccessToken}}</p>
            <ul>
                {{range $key, $value := .Claims}}
                    <li><strong>{{$key}}:</strong> {{$value}}</li>
                {{end}}
            </ul>
            <a href="/logout">Logout</a>
        </body>
    </html>`

	t := template.Must(template.New("claims").Parse(tmpl))

	var buf bytes.Buffer
	if err := t.Execute(&buf, pageData); err != nil {
		return err
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(buf.String())
}
