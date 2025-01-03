package controller

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
)

type OidcDummy struct {
	mu      sync.RWMutex
	clients map[string]Client
}

func (d *OidcDummy) CreateClient(ctx context.Context, req CreateClientRequest) (*Client, error) {
	secret := strconv.Itoa(rand.Int())
	client := Client{
		ClientID:     req.Name,
		ClientSecret: secret,
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.clients[req.Name] = client
	return &client, nil
}

func (d *OidcDummy) GetClient(ctx context.Context, req GetClientRequest) (*Client, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if client, ok := d.clients[req.Name]; ok {
		return &client, nil
	}
	return nil, ErrNotFound
}

func (d *OidcDummy) DeleteClient(ctx context.Context, req DeleteClientRequest) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.clients[req.Name]; ok {
		delete(d.clients, req.Name)
		return nil
	}
	return ErrNotFound
}
