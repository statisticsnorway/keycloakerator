package controller

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v13"
	"golang.org/x/oauth2"
)

type ClientNotFoundError struct {
	ClientId string
}

func (e *ClientNotFoundError) Error() string {
	return fmt.Sprintf("client %q not found", e.ClientId)
}

type GocloakWrapper struct {
	Realm       string
	GoCloak     *gocloak.GoCloak
	TokenSource oauth2.TokenSource
}

var _ Keycloak = (*GocloakWrapper)(nil)

func NewGocloakWrapper(baseUrl, realm string, ts oauth2.TokenSource) *GocloakWrapper {
	return &GocloakWrapper{
		Realm:       realm,
		TokenSource: ts,
		GoCloak:     gocloak.NewClient(baseUrl),
	}
}

func (g *GocloakWrapper) CreateClient(ctx context.Context, client gocloak.Client) (string, error) {
	token, err := g.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}

	return g.GoCloak.CreateClient(ctx, token.AccessToken, g.Realm, client)
}

func (g *GocloakWrapper) GetClient(ctx context.Context, idOfClient string) (*gocloak.Client, error) {
	token, err := g.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	return g.GoCloak.GetClient(ctx, token.AccessToken, g.Realm, idOfClient)
}

func (g *GocloakWrapper) DeleteClient(ctx context.Context, idOfClient string) error {
	token, err := g.TokenSource.Token()
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	return g.GoCloak.DeleteClient(ctx, token.AccessToken, g.Realm, idOfClient)
}

func (g *GocloakWrapper) GetClientByClientId(ctx context.Context, clientId string) (*gocloak.Client, error) {
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

	return nil, &ClientNotFoundError{ClientId: clientId}
}
