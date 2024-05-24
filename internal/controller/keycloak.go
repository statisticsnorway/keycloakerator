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
	GoCloak     *gocloak.GoCloak
	TokenSource oauth2.TokenSource
}

var _ Keycloak = (*GocloakWrapper)(nil)

func (g *GocloakWrapper) CreateClient(ctx context.Context, realm string, client gocloak.Client) (string, error) {
	token, err := g.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}

	return g.GoCloak.CreateClient(ctx, token.AccessToken, realm, client)
}

func (g *GocloakWrapper) GetClient(ctx context.Context, realm, idOfClient string) (*gocloak.Client, error) {
	token, err := g.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	return g.GoCloak.GetClient(ctx, token.AccessToken, realm, idOfClient)
}

func (g *GocloakWrapper) DeleteClient(ctx context.Context, realm, idOfClient string) error {
	token, err := g.TokenSource.Token()
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	return g.GoCloak.DeleteClient(ctx, token.AccessToken, realm, idOfClient)
}

func (g *GocloakWrapper) GetClientByClientId(ctx context.Context, realm string, clientId string) (*gocloak.Client, error) {
	token, err := g.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	clients, err := g.GoCloak.GetClients(ctx, token.AccessToken, realm, gocloak.GetClientsParams{ClientID: &clientId})
	if err != nil {
		return nil, fmt.Errorf("get clients: %w", err)
	}

	for _, client := range clients {
		return client, nil
	}

	return nil, &ClientNotFoundError{ClientId: clientId}
}
