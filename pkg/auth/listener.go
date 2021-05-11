package auth

import (
	"context"
	"encoding/json"
	"github.com/afosto/cli/pkg/client"
	"github.com/afosto/cli/pkg/data"
	"github.com/afosto/cli/pkg/logging"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/browser"
	"io/ioutil"
	"net/http"
	"sync"
)

const (
	LocalAuthServer = "localhost:8888"
)

type loginResource struct {
	wg     *sync.WaitGroup
	server *http.Server
	user   *data.User
}

var user *data.User

func GetUser() *data.User {
	return user
}

func LoadFromStorage() *data.User {
	if data2, err := ioutil.ReadFile("user.json"); err == nil {
		loadedUser := data.User{}
		if err := json.Unmarshal(data2, &loadedUser); err == nil {
			claims := jwt.StandardClaims{}
			parser := jwt.Parser{}
			parser.ParseUnverified(loadedUser.GetAccessToken(), &claims)
			if err := claims.Valid(); err == nil {
				return &loadedUser
			}
		}
	}
	return nil
}

func GetImplicitUser(permissions []string) *data.User {
	err := browser.OpenURL(client.GetAuthorizationURL(permissions))
	if err != nil {
		logging.Log.Fatal("could not call browser")
	}

	resource := getLoginResource()
	go resource.startListener()
	resource.await()

	user = resource.user

	if userData, err := json.Marshal(user); err == nil {
		_ = ioutil.WriteFile("user.json", userData, 0644)
	}
	return user

}

func getLoginResource() *loginResource {
	ll := &loginResource{
		user: &data.User{},
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	ll.wg = &wg
	return ll
}

func (ll *loginResource) startListener() {
	http.HandleFunc("/return", ll.loadUserData)
	ll.server = &http.Server{Addr: LocalAuthServer}
	go ll.server.ListenAndServe()
	logging.Log.Debug("Connecting to Afosto/IO")

}

func (ll *loginResource) await() {
	ll.wg.Wait()
	logging.Log.Debugf("✔ Authenticated as %s (%s) at %s (%s)",
		ll.user.Name, ll.user.Email, ll.user.TenantName, ll.user.TenantID)

	_ = ll.server.Shutdown(context.Background())
}

//Load the userdata onto the login resource
func (ll *loginResource) loadUserData(rw http.ResponseWriter, request *http.Request) {
	defer ll.wg.Done()
	_, err := rw.Write([]byte("Success - you can close this window"))
	if err != nil {
		logging.Log.Fatal(err)
	}

	if f, ok := rw.(http.Flusher); ok {
		f.Flush()
	}

	ll.user.SetAccessToken(request.URL.Query()["access_token"][0])
	idTokenString := request.URL.Query()["id_token"][0]
	accessTokenClaims := jwt.MapClaims{}
	_, _ = jwt.ParseWithClaims(ll.user.GetAccessToken(), accessTokenClaims, func(token *jwt.Token) (interface{}, error) {
		return token, nil
	})
	idTokenClaims := jwt.MapClaims{}
	_, _ = jwt.ParseWithClaims(idTokenString, idTokenClaims, func(token *jwt.Token) (interface{}, error) {
		return token, nil
	})

	for key, val := range idTokenClaims {
		if key == "name" {
			ll.user.Name = val.(string)
		} else if key == "email" {
			ll.user.Email = val.(string)
		}
	}

	for key, val := range accessTokenClaims {
		if key == "tenant" {
			ll.user.TenantID = val.(string)
		} else if key == "sub" {
			ll.user.ID = val.(string)
		}
	}

	tenant, err := client.GetClient(ll.user.TenantID, ll.user.GetAccessToken()).GetTenant()
	if err != nil {
		logging.Log.Fatal("could not fetch tenant")
	}

	ll.user.TenantName = tenant.Name
}
