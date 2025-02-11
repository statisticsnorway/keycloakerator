package keycloak

import (
	"context"

	"github.com/keycloak/terraform-provider-keycloak/keycloak"
	"github.com/statisticsnorway/keycloakerator/internal/controller"
)

type TerraformProviderWrapper struct {
	client *keycloak.KeycloakClient
	realm  string
}

func NewTerraformProviderWrapper(ctx context.Context, url, clientId, clientSecret, realm string) (*TerraformProviderWrapper, error) {
	client, err := keycloak.NewKeycloakClient(
		ctx,
		url, "",
		clientId, clientSecret,
		realm,
		"", "", true, 15, "", false, "Keycloakerator/v0.0.0", false, nil)
	if err != nil {
		return nil, err
	}

	return &TerraformProviderWrapper{client: client, realm: realm}, nil
}

func (k *TerraformProviderWrapper) CreateClient(ctx context.Context, req controller.CreateClientRequest) (*controller.Client, error) {
	keycloakClient := &keycloak.OpenidClient{
		RealmId:           k.realm,
		ClientId:          req.Name,
		Name:              req.Name,
		Enabled:           true,
		PublicClient:      false,
		ValidRedirectUris: req.RedirectURIs,
	}

	if err := k.client.NewOpenidClient(ctx, keycloakClient); err != nil {
		return nil, err
	}

	keycloakClient, err := k.client.GetOpenidClient(ctx, keycloakClient.RealmId, keycloakClient.ClientId)
	if err != nil {
		return nil, err
	}

	if err := k.client.AttachOpenidClientDefaultScopes(ctx, k.realm, keycloakClient.Id, []string{
		"email",
		"profile",
		"roles",
		"web-origins",
	}); err != nil {
		return nil, err
	}

	if err := k.client.NewOpenIdAudienceProtocolMapper(ctx, &keycloak.OpenIdAudienceProtocolMapper{
		AddToIdToken:           true,
		AddToAccessToken:       true,
		IncludedCustomAudience: keycloakClient.ClientId,
		ClientId:               keycloakClient.Id,
	}); err != nil {
		return nil, err
	}

	if err := k.client.NewOpenIdHardcodedClaimProtocolMapper(ctx, &keycloak.OpenIdHardcodedClaimProtocolMapper{
		AddToIdToken:     true,
		AddToAccessToken: true,
		ClientId:         keycloakClient.Id,
		Name:             "dapla-user",
		ClaimName:        "dapla.user",
		ClaimValue:       req.DaplaUser,
	}); err != nil {
		return nil, err
	}

	if err := k.client.NewOpenIdHardcodedClaimProtocolMapper(ctx, &keycloak.OpenIdHardcodedClaimProtocolMapper{
		AddToIdToken:     true,
		AddToAccessToken: true,
		ClientId:         keycloakClient.Id,
		Name:             "dapla-group",
		ClaimName:        "dapla.group",
		ClaimValue:       req.DaplaGroup,
	}); err != nil {
		return nil, err
	}

	return &controller.Client{
		ClientID:     keycloakClient.ClientId,
		ClientSecret: keycloakClient.ClientSecret,
	}, nil
}

func (k *TerraformProviderWrapper) DeleteClient(ctx context.Context, req controller.DeleteClientRequest) error {
	client, err := k.client.GetOpenidClientByClientId(ctx, k.realm, req.Name)
	if err != nil {
		return err
	}

	return k.client.DeleteOpenidClient(ctx, k.realm, client.Id)
}

func (k *TerraformProviderWrapper) GetClient(ctx context.Context, req controller.GetClientRequest) (*controller.Client, error) {
	client, err := k.client.GetOpenidClientByClientId(ctx, k.realm, req.Name)
	if err != nil {
		return nil, err
	}

	return &controller.Client{
		ClientID:     client.ClientId,
		ClientSecret: client.ClientSecret,
	}, nil
}
