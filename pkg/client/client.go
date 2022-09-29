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
	"strings"
	"time"
)

const (
	BaseAuthorizationURL = "https://afosto.app/auth/authorize"
	BaseApiUrl           = "https://afosto.app/api"
	OauthClientID        = "51403354ded11942d7195c66b9e81f71b74f56cd8adc539277823e179da8"
	RedirectURL          = "http://localhost:8888/return"
)

var (
	cl *AfostoClient
)

type tripper struct {
	tenantID    string
	accessToken string
	rt          http.RoundTripper
}

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

type SignatureRequest struct {
	IsPublic bool              `json:"is_public"`
	Path     string            `json:"path"`
	IsListed bool              `json:"is_listed"`
	Method   string            `json:"method"`
	Metadata map[string]string `json:"metadata"`
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
			c:           cache.New(time.Minute*5, time.Minute),
		}
		cl.client = &http.Client{
			Timeout: time.Second * 30,
			Transport: &tripper{
				accessToken: accessToken,
				tenantID:    tenantID,
				rt:          http.DefaultTransport,
			},
		}
	}

	return cl
}

func (ac *AfostoClient) GetTenant() (*data.Tenant, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", BaseApiUrl, "iam/tenants/"+ac.tenantID), nil)
	var tenant data.Tenant
	b, _, err := handle(ac.client.Do(req))
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(b, &tenant)

	return &tenant, nil

}

func (ac *AfostoClient) GetSignature(dir string, method string, asPrivateDirectory bool) (string, error) {
	tenant, err := ac.GetTenant()
	if err != nil {
		return "", err
	}
	result, ok := ac.c.Get(tenant.ID + dir)
	var signature string
	if !ok {

		type request struct {
			Data SignatureRequest `json:"data"`
		}

		signatureRequest := request{
			Data: SignatureRequest{
				IsPublic: !asPrivateDirectory,
				IsListed: true,
				Path:     dir,
				Method:   method,
			},
		}

		req, _ := http.NewRequest("POST", fmt.Sprintf("%s/%s", BaseApiUrl, "storage/files/signature"), jsonPayload(signatureRequest))
		req.Header.Set("content-type", "application/json")

		type response struct {
			Data data.Signature `json:"data"`
		}

		var signatureResponse response
		b, _, err := handle(ac.client.Do(req))
		if err != nil {
			return "", err
		}

		if err := json.Unmarshal(b, &signatureResponse); err != nil {
			return "", err
		}

		signature = signatureResponse.Data.Signature
		ac.c.Set(tenant.ID+dir, signature, cache.DefaultExpiration)

	} else {
		signature = result.(string)
	}
	return signature, nil
}

func (ac *AfostoClient) ListDirectory(dir string, cursor string) ([]data.File, string, error) {

	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", BaseApiUrl, "cnt/files?dir="+dir), nil)
	if cursor != "" {
		req.Header.Set("x-page", cursor)
	}
	req.Header.Set("x-page-size", "25")
	files := []data.File{}
	b, headers, err := handle(ac.client.Do(req))
	if err != nil {
		return nil, "", err
	}
	_ = json.Unmarshal(b, &files)

	var cursorResponse string
	if v, ok := headers["X-Page"]; ok {
		cursorResponse = v[0]
	}

	return files, cursorResponse, nil
}

func (ac *AfostoClient) ListDirectories(dir string) ([]string, error) {

	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/%s", BaseApiUrl, "cnt/directories"), nil)

	directories := struct {
		Directories []string `json:"directories"`
	}{}

	resp, err := ac.client.Do(req)

	_ = resp
	b, _, err := handle(resp, err)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(b, &directories)

	list := []string{}

	for _, directory := range directories.Directories {
		if strings.HasPrefix(directory, dir) {
			list = append(list, directory)
		}
	}
	return list, nil
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

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/%s", BaseApiUrl, "storage/files/upload/"+signature), body)
	req.Header.Set("content-type", writer.FormDataContentType())

	type response struct {
		Data []data.File `json:"data"`
	}

	var fileResponse response
	b, _, err := handle(ac.client.Do(req))
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(b, &fileResponse)

	if len(fileResponse.Data) > 0 {
		return &fileResponse.Data[0], nil
	}

	return nil, errors.New("got wrong response")
}

func (ac *AfostoClient) Download(url *url.URL) ([]byte, error) {
	req, _ := http.NewRequest("GET", url.String(), nil)
	b, _, err := handle(ac.client.Do(req))
	return b, err
}

func (ac *AfostoClient) Query(query string, parameters interface{}) (*QueryResult, error) {
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/%s", BaseApiUrl, "gql"), jsonPayload(Query{
		OperationName: nil,
		Query:         query,
		Variables:     parameters,
	}))

	req.Header.Set("content-type", "application/json")

	var result QueryResult
	b, _, err := handle(ac.client.Do(req))
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(b, &result)

	return &result, nil

}

func (ac *tripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request.Header.Set("authorization", "Bearer "+ac.accessToken)

	return ac.rt.RoundTrip(request)

}

func jsonPayload(payload interface{}) *bytes.Reader {
	b, err := json.Marshal(payload)
	if err != nil {
		logging.Log.Fatal(err)
	}
	return bytes.NewReader(b)
}

func handle(res *http.Response, err error) ([]byte, map[string][]string, error) {
	if err != nil {
		return nil, nil, err
	}

	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}
	headers := map[string][]string{}
	for key, values := range res.Header {
		headers[key] = values
	}

	return b, headers, nil
}
