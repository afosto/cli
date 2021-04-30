package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/dghubble/sling"
	"github.com/patrickmn/go-cache"
	"io"
	"mime/multipart"
	"strconv"
	"strings"
	"time"
)

var (
	Jwt = ctxKey{Name: "jwt"}
)

type ctxKey struct {
	Name string
}

const (
	FileServiceUrl = "https://api.afosto.io/cnt"
)

type FileClient struct {
	sling *sling.Sling
	cache *cache.Cache
}

type File struct {
	ID        string            `json:"id"`
	Filename  string            `json:"filename"`
	Label     string            `json:"label"`
	Dir       string            `json:"dir"`
	Type      string            `json:"type"`
	Mime      string            `json:"mime"`
	Url       string            `json:"url"`
	IsPublic  bool              `json:"is_public"`
	IsListed  bool              `json:"is_listed"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type Signature struct {
	ExpiresAt time.Time `json:"expires_at"`
	Signature string    `json:"signature"`
}

func NewFileClient() *FileClient {
	c := FileClient{
		sling: sling.New().Base(FileServiceUrl),
		cache: cache.New(time.Hour, time.Hour),
	}

	return &c
}

func (ic *FileClient) prepareRequest(ctx context.Context) (*sling.Sling, error) {
	request := ic.sling.New()

	if token, ok := ctx.Value(Jwt).(string); !ok {
		return nil, errors.New("jwt is missing from context")
	} else {
		request = request.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	}

	return request, nil
}

func (c *FileClient) GetSignedURL(ctx context.Context, fileID string) (string, error) {
	request, err := c.prepareRequest(ctx)
	if err != nil {
		return "", err
	}
	r := request.Get("/cnt/files/" + fileID + "/url")

	responseStruct := struct {
		URL string `json:"url"`
	}{}

	response, err := r.ReceiveSuccess(&responseStruct)
	if err != nil {
		return "", err
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", errors.New("got " + strconv.Itoa(response.StatusCode))
	}

	return responseStruct.URL, nil

}

func (c *FileClient) GetSignature(ctx context.Context, dir string, method string) (string, error) {

	result, ok := c.cache.Get(getKey(ctx.Value(Jwt).(string), dir))
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
		r := c.sling.New().Post("/cnt/files/signature").BodyJSON(&signatureRequest)
		signatureResponse := Signature{}

		_, err := r.ReceiveSuccess(&signatureResponse)
		if err != nil {
			return "", err
		}
		signature = signatureResponse.Signature
		c.cache.Set(getKey(ctx.Value(Jwt).(string), dir), signature, cache.DefaultExpiration)
	} else {
		signature = result.(string)
	}

	return signature, nil
}

func (c *FileClient) GetPublicSignature(ctx context.Context, dir string, method string) (string, error) {
	result, ok := c.cache.Get(getKey(ctx.Value(Jwt).(string), dir))
	var signature string
	if !ok {
		signatureRequest := struct {
			IsPublic bool              `json:"is_public"`
			Path     string            `json:"path"`
			IsListed bool              `json:"is_listed"`
			Method   string            `json:"method"`
			Metadata map[string]string `json:"metadata"`
		}{
			IsPublic: true,
			IsListed: true,
			Path:     dir,
			Method:   method,
		}

		r, err := c.prepareRequest(ctx)
		if err != nil {
			return "", err
		}

		signatureResponse := Signature{}
		if resp, err := r.Post("/cnt/files/signature").BodyJSON(signatureRequest).ReceiveSuccess(&signatureResponse); err != nil {
			return "", err
		} else if resp.StatusCode > 299 || resp.StatusCode < 200 {
			return "", errors.New("invalid statuscode")
		}

		signature = signatureResponse.Signature
		c.cache.Set(getKey(ctx.Value(Jwt).(string), dir), signature, cache.DefaultExpiration)
	} else {
		signature = result.(string)
	}

	return signature, nil
}

func (c *FileClient) Upload(ctx context.Context, source io.ReadCloser, labelFilename string, signature string) (*File, error) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", labelFilename)

	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(part, source); err != nil {
		return nil, err
	}

	writer.Close()
	req, err := c.prepareRequest(ctx)
	if err != nil {
		return nil, err
	}

	req = req.Post("/cnt/files/upload/"+signature).Body(body).Add("Content-Type", writer.FormDataContentType())

	var fileResponse []File
	res, err := req.ReceiveSuccess(&fileResponse)
	if err != nil {
		return nil, err
	}

	if res.StatusCode > 299 || res.StatusCode < 200 {
		return nil, errors.New("invalid statuscode")
	}

	return &fileResponse[0], nil

}

func getKey(t string, dir string) string {
	return strings.Join([]string{t, dir}, "-")
}
