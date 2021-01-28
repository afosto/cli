package auth

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gosuri/uilive"
	"github.com/pkg/browser"
	"log"
	"net/http"
	"sync"
	"time"
)

type tenant struct {
	id   string
	name string
}

type user struct {
	id          string
	name        string
	email       string
	accessToken string
}

type client struct {
	clientID     string
	clientSecret string
}

type loginResource struct {
	wg     *sync.WaitGroup
	client *http.Client
	tenant tenant
	user   user
}

func Login() {
	listener := &loginResource{
		client: &http.Client{Timeout: time.Second * 5},
	}
	openBrowser()
	listener.catchResponse()

	log.Print(listener.tenant.id)

}

func (listener *loginResource) searchClient() (*client, error) {
	//listener.client.Get("")
	return nil, nil
}

func openBrowser() {
	redirectURL := "http://localhost:8888/return"
	clientID := "51403354ded11942d7195c66b9e81f71b74f56cd8adc539277823e179da8"
	url := "https://login.afosto.io/authorize?client_id=" + clientID + "&redirect_uri=" +
		redirectURL + "&response_type=token%20id_token&scope=openid%20email%20profile%20app%3Aintegrations%3Aread%20app%3Aintegrations%3Awrite"

	err := browser.OpenURL(url)
	if err != nil {
		log.Fatal("could not call browser")
	}
}

func (listener *loginResource) catchResponse() {
	writer := uilive.New()
	writer.Start()
	wg := sync.WaitGroup{}
	wg.Add(1)
	listener.wg = &wg
	go listener.start(writer)
	listener.wg.Wait()
	writer.Stop()

}

func (ll *loginResource) start(writer *uilive.Writer) {
	http.HandleFunc("/return", ll.returnToken)
	server := &http.Server{Addr: "localhost:8888"}
	go server.ListenAndServe()
	icons := []string{"/", "-", "\\", "|", "/", "-", "\\", "|"}
	k := 0
	for {
		time.Sleep(300 * time.Millisecond)
		if k == len(icons) {
			k = 0
		}
		_, _ = fmt.Fprintln(writer, "Connecting to Afosto/IO [ "+icons[k]+" ]")
		k++
	}

}
func (ll *loginResource) returnToken(rw http.ResponseWriter, request *http.Request) {
	defer ll.wg.Done()
	_, err := rw.Write([]byte("Success - you can close this window"))
	if err != nil {
		log.Fatal(err)
	}

	if f, ok := rw.(http.Flusher); ok {
		f.Flush()
	}

	ll.user.accessToken = request.URL.Query()["access_token"][0]
	idTokenString := request.URL.Query()["id_token"][0]
	accessTokenClaims := jwt.MapClaims{}
	_, _ = jwt.ParseWithClaims(ll.user.accessToken, accessTokenClaims, func(token *jwt.Token) (interface{}, error) {
		return token, nil
	})
	idTokenClaims := jwt.MapClaims{}
	_, _ = jwt.ParseWithClaims(idTokenString, idTokenClaims, func(token *jwt.Token) (interface{}, error) {
		return token, nil
	})

	for key, val := range idTokenClaims {
		if key == "name" {
			ll.user.name = val.(string)
		} else if key == "email" {
			ll.user.email = val.(string)
		}
	}

	for key, val := range accessTokenClaims {
		if key == "tenant" {
			ll.tenant.id = val.(string)
		} else if key == "sub" {
			ll.user.id = val.(string)
		}
	}
}
