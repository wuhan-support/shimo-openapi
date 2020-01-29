package shimo_openapi

import "time"

type Client struct {
	clientId string
	clientSecret string
	username string
	password string
	scope string

	credential struct {
		accessToken string
		accessTokenExpiresAt time.Time

		refreshToken string
	}
}

type oAuthResponse struct {
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Opts struct {
	SheetName string `json:"sheet_name"`
	EndRow int `json:"end_row"`
	EndCol string `json:"end_col"`
}