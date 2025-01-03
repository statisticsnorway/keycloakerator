package keycloak

import (
	"context"
	"errors"
	"fmt"

	"github.com/Nerzal/gocloak/v13"
	"github.com/statisticsnorway/keycloakerator/internal/controller"
	"golang.org/x/oauth2"
	"k8s.io/utils/ptr"
)

type GocloakWrapper struct {
	Realm       string
	GoCloak     *gocloak.GoCloak
	TokenSource oauth2.TokenSource
}

var ErrNotFound error = errors.New("client not found")

func NewGocloakWrapper(baseUrl, realm string, ts oauth2.TokenSource) *GocloakWrapper {
	return &GocloakWrapper{
		Realm:       realm,
		TokenSource: ts,
		GoCloak:     gocloak.NewClient(baseUrl),
	}
}

func (g *GocloakWrapper) CreateClient(ctx context.Context, req controller.CreateClientRequest) (*controller.Client, error) {
	token, err := g.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("get keycloak token: %w", err)
	}

	client := &gocloak.Client{
		ClientID:     &req.Name,
		Name:         &req.Name,
		Enabled:      ptr.To(true),
		PublicClient: ptr.To(false),
		RedirectURIs: &req.RedirectURIs,
		DefaultClientScopes: &[]string{
			"email",
			"profile",
			"roles",
			"web-origins",
		},
	}

	internalId, err := g.GoCloak.CreateClient(ctx, token.AccessToken, g.Realm, *client)
	if err != nil {
		return nil, fmt.Errorf("create keycloak client: %w", err)
	}

	client, err = g.getClientByInternalId(ctx, internalId)
	if err != nil {
		return nil, fmt.Errorf("get new keycloak client: %w", err)
	}

	clientAudience := gocloak.ProtocolMapperRepresentation{
		Name:           ptr.To(""),
		Protocol:       ptr.To("openid-connect"),
		ProtocolMapper: ptr.To("oidc-audience-mapper"),
		Config: &map[string]string{
			"id.token.claim":           "true",
			"access.token.claim":       "true",
			"included.custom.audience": *client.ClientID,
		},
	}

	if _, err = g.createClientProtocolMapper(ctx, internalId, clientAudience); err != nil {
		return nil, fmt.Errorf("create audience mapper: %w", err)
	}

	return &controller.Client{
		ClientID:     *client.ClientID,
		ClientSecret: *client.Secret,
	}, nil
}

func (g *GocloakWrapper) createClientProtocolMapper(ctx context.Context, clientInternalId string, mapper gocloak.ProtocolMapperRepresentation) (string, error) {
	token, err := g.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}

	return g.GoCloak.CreateClientProtocolMapper(ctx, token.AccessToken, g.Realm, clientInternalId, mapper)
}

func (g *GocloakWrapper) getClientByInternalId(ctx context.Context, internalId string) (*gocloak.Client, error) {
	token, err := g.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	return g.GoCloak.GetClient(ctx, token.AccessToken, g.Realm, internalId)
}

func (g *GocloakWrapper) DeleteClient(ctx context.Context, req controller.DeleteClientRequest) error {
	token, err := g.TokenSource.Token()
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	client, err := g.getClientByClientId(ctx, req.Name)
	if err != nil {
		return fmt.Errorf("get client: %w", err)
	}

	return g.GoCloak.DeleteClient(ctx, token.AccessToken, g.Realm, *client.ID)
}

func (g *GocloakWrapper) getClientByClientId(ctx context.Context, clientId string) (*gocloak.Client, error) {
	token, err := g.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	clients, err := g.GoCloak.GetClients(ctx, token.AccessToken, g.Realm, gocloak.GetClientsParams{ClientID: &clientId})
	if err != nil {
		return nil, fmt.Errorf("get clients: %w", err)
	}

	for _, client := range clients {
		return client, nil
	}

	return nil, ErrNotFound
}

func (g *GocloakWrapper) GetClient(ctx context.Context, req controller.GetClientRequest) (*controller.Client, error) {
	client, err := g.getClientByClientId(ctx, req.Name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, controller.ErrNotFound
		}
		return nil, fmt.Errorf("get client: %w", err)
	}

	return &controller.Client{
		ClientID:     *client.ClientID,
		ClientSecret: *client.Secret,
	}, nil
}
