package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/afosto/cli/pkg/data"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	BaseAuthorizationURL = "https://login.afosto.io/authorize"
	BaseApiUrl           = "https://api.afosto.io"
	OauthClientID        = "51403354ded11942d7195c66b9e81f71b74f56cd8adc539277823e179da8"
	RedirectURL          = "http://localhost:8888/return"
)

var (
	cl *AfostoClient
)

type AfostoClient struct {
	client      *http.Client
	tenantID    string
	accessToken string
}

type Query struct {
	OperationName *string     `json:"operationName"`
	Query         string      `json:"query"`
	Variables     interface{} `json:"variables"`
}

type QueryResult struct {
	Data   map[string]interface{} `json:"data"`
	Errors []QueryError           `json:"errors"`
}

type QueryError struct {
	Message string   `json:"message"`
	Path    []string `json:"path"`
}

func GetAuthorizationURL() string {
	scopes := []string{
		"openid",
		"email",
		"profile",
		"cnt:index:read",
		"iam:users:read",
		"iam:roles:read",
		"iam:tenants:read",
		"lcs:locations:read",
		"lcs:handling:read",
		"lcs:shipments:read",
		"odr:orders:read",
		"odr:coupons:read",
		"odr:invoices:read",
		"rel:contacts:read",
		"rel:identity:read",
	}

	return fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=token+id_token&scope=%s",
		BaseAuthorizationURL, OauthClientID, url.QueryEscape(RedirectURL), url.QueryEscape(strings.Join(scopes, " ")))
}

func GetClient(tenantID string, accessToken string) *AfostoClient {
	if cl == nil {
		cl = &AfostoClient{
			tenantID:    tenantID,
			accessToken: accessToken,
			client: &http.Client{
				Timeout: time.Second * 5,
			},
		}
	}

	return cl
}

func (ac *AfostoClient) GetTenant() (*data.Tenant, error) {
	b, err := ac.request("GET", "iam/tenants/"+ac.tenantID, nil)
	if err != nil {
		return nil, err
	}

	var tenant data.Tenant
	_ = json.Unmarshal(b, &tenant)

	return &tenant, nil

}

func (ac *AfostoClient) Query(query string, parameters interface{}) (*QueryResult, error) {
	b, err := ac.request("POST", "gql", Query{
		OperationName: nil,
		Query:         query,
		Variables:     parameters,
	})
	if err != nil {
		return nil, err
	}

	var result QueryResult
	_ = json.Unmarshal(b, &result)

	return &result, nil

}

func (ac *AfostoClient) request(method string, path string, payload interface{}) ([]byte, error) {
	var request *http.Request
	var err error

	if payload == nil {
		request, err = http.NewRequest(method, fmt.Sprintf("%s/%s", BaseApiUrl, path), nil)
	} else {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		request, err = http.NewRequest(method, fmt.Sprintf("%s/%s", BaseApiUrl, path), bytes.NewReader(b))
	}
	if err != nil {
		return nil, err
	}
	request.Header.Set("authorization", "Bearer "+ac.accessToken)
	request.Header.Set("content-type", "application/json")

	res, err := ac.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return b, nil

}
