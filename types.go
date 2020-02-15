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
	// the suffix in the header to be removed
	HeaderSuffix    string `json:"header_suffix"`
	CacheTTL time.Duration `json:"cache_ttl"`
}

type WriteOpts struct {
	Range string `json:"range"`
	Resource *WriteResource `json:"resource"`
}

type WriteResource struct {
	Values [][]interface{} `json:"values"`
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

// Because the map data is unordered, 
// you need to customize the order in which the fields are stored
type WriteObj interface {
	Values() []interface{}
}

func NewWriteOpts(sheetName string, v WriteObj) (*WriteOpts) {
	w := &WriteOpts{ Range: sheetName, Resource: &WriteResource{} }
	w.Resource.Values = make([][]interface{}, 0)
	w.Resource.Values = append(w.Resource.Values, v.Values())
	return w
}
