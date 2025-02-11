package controller

import (
	"context"
	"errors"
)

type OIDCService interface {
	CreateClient(context.Context, CreateClientRequest) (*Client, error)
	GetClient(context.Context, GetClientRequest) (*Client, error)
	DeleteClient(context.Context, DeleteClientRequest) error
}

var ErrNotFound error = errors.New("client not found")

type Client struct {
	ClientID     string
	ClientSecret string
}

type CreateClientRequest struct {
	Name         string
	DaplaUser    string
	DaplaGroup   string
	RedirectURIs []string
}

type GetClientRequest struct {
	Name string
}

type DeleteClientRequest struct {
	Name string
}
