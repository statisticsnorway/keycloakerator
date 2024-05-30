package controller

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/Nerzal/gocloak/v13"
)

var _ Keycloak = (*KeycloakDummy)(nil)

type KeycloakDummy struct {
	clients []gocloak.Client
}

func (d *KeycloakDummy) CreateClient(ctx context.Context, newClient gocloak.Client) (string, error) {
	id := strconv.Itoa(len(d.clients))
	newClient.ID = &id
	hash := md5.Sum([]byte(fmt.Sprintf("%s-%s", id, *(newClient.Name))))
	secret := hex.EncodeToString(hash[:])
	newClient.Secret = &secret
	d.clients = append(d.clients, newClient)
	return id, nil
}

func (d *KeycloakDummy) GetClientByClientId(ctx context.Context, clientId string) (*gocloak.Client, error) {
	for _, client := range d.clients {
		if *client.ClientID == clientId {
			return &client, nil
		}
	}
	return nil, &ClientNotFoundError{ClientId: clientId}
}

func (d *KeycloakDummy) GetClient(ctx context.Context, idOfClient string) (*gocloak.Client, error) {
	id, err := strconv.Atoi(idOfClient)
	if err != nil {
		return nil, err
	}
	if id < 0 || id >= len(d.clients) {
		return nil, fmt.Errorf("invalid dummy client ID %d", id)
	}
	return &d.clients[id], nil
}

func (d *KeycloakDummy) DeleteClient(ctx context.Context, idOfClient string) error {
	id, err := strconv.Atoi(idOfClient)
	if err != nil {
		return err
	}
	if id < 0 || id >= len(d.clients) {
		return errors.New("invalid dummy client ID")
	}
	d.clients = append(d.clients[:id], d.clients[id+1:]...)
	return nil
}
