package shimo_openapi

import (
	"sync"
	"time"
)

// Client provides a way to interact with Shimo.im using such credentials
type Client struct {
	clientId     string
	clientSecret string
	username     string
	password     string
	scope        string
	asyncSign    chan sign
	l            sync.RWMutex
	cache        map[string]*Cache

	credential struct {
		accessToken          string
		accessTokenExpiresAt time.Time

		refreshToken string
	}
}

type oAuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Opts provide options for getting sheet data
type Opts struct {
	SheetName string `json:"sheet_name"`
	EndRow    int    `json:"end_row"`
	EndCol    string `json:"end_col"`
}

type sign struct {
	Opts
	FileID string
}

// Cache cache request data
type Cache struct {
	Opts
	expire time.Time
	result []byte
}
