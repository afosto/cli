package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/afosto/cli/pkg/data"
	"github.com/afosto/cli/pkg/logging"
	"github.com/patrickmn/go-cache"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
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
	c           *cache.Cache
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

func GetAuthorizationURL(scopes []string) string {
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
			c: cache.New(time.Minute*5, time.Minute),
		}
	}

	return cl
}

func (ac *AfostoClient) GetTenant() (*data.Tenant, error) {
	b, err := ac.request("GET", "iam/tenants/"+ac.tenantID, nil, "application/json")
	if err != nil {
		return nil, err
	}

	var tenant data.Tenant
	_ = json.Unmarshal(b, &tenant)

	return &tenant, nil

}

func (ac *AfostoClient) GetSignature(dir string, method string) (string, error) {
	tenant, err := ac.GetTenant()
	if err != nil {
		return "", err
	}
	result, ok := ac.c.Get(tenant.ID + dir)
	var signature string
	if !ok {
		signatureRequest := struct {
			IsPublic bool              `json:"is_public"`
			Path     string            `json:"path"`
			IsListed bool              `json:"is_listed"`
			Method   string            `json:"method"`
			Metadata map[string]string `json:"metadata"`
		}{
			IsPublic: false,
			IsListed: true,
			Path:     dir,
			Method:   method,
		}
		resultData, err := ac.request("POST", "cnt/files/signature", signatureRequest, "application/json")
		signatureResponse := data.Signature{}

		err = json.Unmarshal(resultData, &signatureResponse)
		if err != nil {
			return "", err
		}

		signature = signatureResponse.Signature
		ac.c.Set(tenant.ID+dir, signature, cache.DefaultExpiration)

	} else {
		signature = result.(string)
	}
	return signature, nil
}

func (ac *AfostoClient) Upload(sourceFilePath string, labelFilename string, signature string) (*data.File, error) {
	file, err := os.Open(sourceFilePath)
	if err != nil {
		logging.Log.Error(err)
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", labelFilename)
	if err != nil {
		logging.Log.Error(err)
		return nil, err
	}
	_, _ = io.Copy(part, file)
	_ = writer.Close()

	result, err := ac.request("POST", "cnt/files/upload/"+signature, body, writer.FormDataContentType())
	if err != nil {
		logging.Log.Error(err)
		return nil, err
	}
	var fileResponse []data.File
	err = json.Unmarshal(result, &fileResponse)
	if err != nil {
		logging.Log.Error(err)
		return nil, err
	}

	if len(fileResponse) > 0 {
		return &fileResponse[0], nil
	}

	return nil, errors.New("got wrong response")

}

func (ac *AfostoClient) Query(query string, parameters interface{}) (*QueryResult, error) {
	b, err := ac.request("POST", "gql", Query{
		OperationName: nil,
		Query:         query,
		Variables:     parameters,
	}, "application/json")
	if err != nil {
		return nil, err
	}

	var result QueryResult
	_ = json.Unmarshal(b, &result)

	return &result, nil

}

func (ac *AfostoClient) request(method string, path string, payload interface{}, contentType string) ([]byte, error) {
	var request *http.Request
	var err error
	if payload == nil {
		request, err = http.NewRequest(method, fmt.Sprintf("%s/%s", BaseApiUrl, path), nil)
	} else {
		switch payload.(type) {
		case *bytes.Buffer:
			request, err = http.NewRequest(method, fmt.Sprintf("%s/%s", BaseApiUrl, path), payload.(*bytes.Buffer))
		case interface{}:
			b, err := json.Marshal(payload)
			if err != nil {
				return nil, err
			}
			request, err = http.NewRequest(method, fmt.Sprintf("%s/%s", BaseApiUrl, path), bytes.NewReader(b))
		default:
			logging.Log.Error("could not build request for " + reflect.TypeOf(payload).String())
			return nil, errors.New("invalid request payload")
		}
	}
	if err != nil {
		return nil, err
	}
	request.Header.Set("authorization", "Bearer "+ac.accessToken)
	request.Header.Set("content-type", contentType)

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
