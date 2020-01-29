package shimo_openapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/wuhan-support/shimo-openapi/transform"
)

const Endpoint = "https://api.shimo.im"
const TokenTTL = time.Hour

var client = http.Client{
	Timeout: time.Minute,
}

// NewClient initializes a new Client
func NewClient(clientId string, clientSecret string, username string, password string, scope string) *Client {
	client := &Client{
		clientId:     clientId,
		clientSecret: clientSecret,
		username:     username,
		password:     password,
		scope:        scope,
		asyncSign:    make(chan sign, 10),
		cache:        make(map[string]*Cache),
	}

	go client.receiveSign()

	return client
}

// doOAuth receives oauth parameters, sends an OAuth request to the server, and returns the access key it got
func (c *Client) doOAuth(v url.Values) (string, error) {
	buf := bytes.NewBufferString(v.Encode())
	req, err := http.NewRequest("POST", Endpoint+"/oauth/token", buf)
	if err != nil {
		return "", nil
	}
	basic := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.clientId, c.clientSecret)))
	//spew.Dump(basic)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", basic))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	//spew.Dump(req)

	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// spew.Dump(response.Body)

	if response.StatusCode != 200 {
		i, _ := ioutil.ReadAll(response.Body)
		return "", fmt.Errorf("non-200 response received when getting token: %v", i)
	}

	var oauthCredentials oAuthResponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&oauthCredentials)
	if err != nil {
		return "", err
	}

	// spew.Dump(oauthCredentials)

	c.credential.accessToken = oauthCredentials.AccessToken
	c.credential.accessTokenExpiresAt = time.Now().Add(TokenTTL)
	c.credential.refreshToken = oauthCredentials.RefreshToken

	return oauthCredentials.AccessToken, nil
}

// getAccessToken uses the credentials to get a new token from server
func (c *Client) getAccessToken() (string, error) {
	v := url.Values{}
	v.Add("grant_type", "password")
	v.Add("scope", c.scope)
	v.Add("username", c.username)
	v.Add("password", c.password)

	return c.doOAuth(v)
}

// refreshToken uses the existing refreshToken to refresh a token
func (c *Client) refreshToken() (string, error) {
	// if there's no refreshtoken we will get the access token again
	if c.credential.refreshToken == "" {
		return c.getAccessToken()
	}

	v := url.Values{}
	v.Add("grant_type", "refresh_token")
	v.Add("scope", c.scope)
	v.Add("refresh_token", c.credential.refreshToken)

	return c.doOAuth(v)
}

// token returns an access token, which such token will be refreshed if it has expired, or it will be
// asked for authorization if there's no access token at all
func (c *Client) token() (string, error) {
	if c.credential.accessToken != "" {
		// have access token; expiration unknown
		if c.credential.accessTokenExpiresAt.After(time.Now()) {
			// have access token && not expired
			return c.credential.accessToken, nil
		} else {
			// have access token && expired
			return c.refreshToken()
		}
	} else {
		// dont have access token
		return c.getAccessToken()
	}
}

// request sends request with token
func (c *Client) request(r *http.Request) ([]byte, error) {
	token, err := c.token()
	if err != nil {
		return nil, err
	}

	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	response, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		i, _ := ioutil.ReadAll(response.Body)
		return nil, fmt.Errorf("non-200 response received when getting docs: %v", string(i))
	}
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed:%v", err)
	}
	return b, nil
}

func (c *Client) getFileFromAPI(fileID string, opts Opts) ([]byte, error) {
	u := path.Join("/files", fileID, "sheets/values")
	u = fmt.Sprintf("%s%s?range=%s!A1:%s%d", Endpoint, u, url.PathEscape(opts.SheetName), opts.EndCol, opts.EndRow)

	request, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.request(request)
	if err != nil {
		return nil, err
	}
	return transform.Transform(resp)
}

func (c *Client) receiveSign() {
	for sign := range c.asyncSign {
		c.updateHandle(sign.FileID, sign.Opts)
	}
}

func (c *Client) updateHandle(fileID string, opts Opts) error {
	c.l.RLock()
	cache, ok := c.cache[fileID]
	c.l.RUnlock()
	if ok &&
		cache.expire.After(time.Now()) &&
		opts.SheetName == cache.SheetName &&
		opts.EndRow == cache.EndRow &&
		opts.EndCol == cache.EndCol {

		fmt.Println("updateHandle repeat")
		return nil
	}

	_, err := c.updateOrCreateCache(fileID, opts)
	return err
}

func (c *Client) updateOrCreateCache(fileID string, opts Opts) (*Cache, error) {
	r, err := c.getFileFromAPI(fileID, opts)
	if err != nil {
		return nil, err
	}

	cache := &Cache{
		expire: time.Now().Add(time.Second * 60),
		Opts:   opts,
		result: r,
	}

	c.l.Lock()
	defer c.l.Unlock()

	c.cache[fileID] = cache
	return cache, nil
}

// GetFileWithOpts gets a file from shimo.im with the specified fileID and Opts. It returns the response io.Reader which can be used to stream responses. The one using this method SHOULD cache the file content response from this method due to the limitation of shimo.im's API.
func (c *Client) GetFileWithOpts(fileID string, opts Opts) ([]byte, error) {
	c.l.RLock()
	cache, ok := c.cache[fileID]
	c.l.RUnlock()
	if !ok {
		cache, err := c.updateOrCreateCache(fileID, opts)
		if err != nil {
			return nil, err
		}
		return cache.result, nil
	}

	if cache.expire.After(time.Now()) &&
		opts.SheetName == cache.SheetName &&
		opts.EndRow == cache.EndRow &&
		opts.EndCol == cache.EndCol {

		return cache.result, nil
	}

	select {
	case c.asyncSign <- sign{FileID: fileID, Opts: opts}:
	case <-time.After(time.Millisecond * 10):
		fmt.Println("send async update sign timeout")
	}

	return cache.result, nil
}

func (c *Client) Close() {
	close(c.asyncSign)
}
