package render

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/afosto/cli/pkg/client"
	"github.com/afosto/cli/pkg/data"
	"github.com/afosto/cli/pkg/logging"
	"github.com/flosch/pongo2/v4"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	AfostoConfigFile = "afosto.config.yml"
)

var (
	ErrorConfigFileNotFound = errors.New("could not find config file")
	ErrorPongoLoader        = errors.New("could not start load files in pongo")
	ErrorTemplateLoader     = errors.New("could not load template")
)

type developmentServer struct {
	hostname       string
	configFilePath string
	port           int
	server         *http.Server
	client         *client.AfostoClient
	pongo          *pongo2.TemplateSet
	entries        map[string]data.Route
	definition     *data.TemplateDefinition
}

func GetDevelopmentServer(hostname string, port int, path string, user *data.User) (*developmentServer, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, ErrorConfigFileNotFound
	}

	loader, err := pongo2.NewSandboxedFilesystemLoader(filepath.Dir(path))
	if err != nil {
		return nil, ErrorPongoLoader
	}
	err = loader.SetBaseDir(filepath.Dir(path))
	if err != nil {
		return nil, ErrorPongoLoader
	}

	devServer := &developmentServer{
		hostname:       hostname,
		port:           port,
		configFilePath: path,
		pongo:          pongo2.NewSet("templates", loader),
		client:         client.GetClient(user.TenantID, user.GetAccessToken()),
	}

	return devServer, nil
}

func (ds *developmentServer) Start() {
	r := mux.NewRouter()
	r.HandleFunc("/index", ds.index).Methods("GET")

	r.HandleFunc("/assets/{file}", ds.serveAsset).Methods("GET")
	r.Handle("/assets/{file}", http.StripPrefix("/assets/", http.FileServer(http.Dir(filepath.Dir(ds.configFilePath)+"/dist/"))))
	r.HandleFunc("/render/{category}/{path}/{id}", ds.render).Methods("GET")
	r.HandleFunc("/render/{category}/{path}", ds.render).Methods("GET")

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", ds.hostname, ds.port),
		Handler: r,
	}
	ds.server = server
	err := ds.server.ListenAndServe()
	if err != nil {
		logging.Log.Fatal(err)
	}
}

func (ds *developmentServer) Stop() {
	_ = ds.server.Shutdown(context.Background())
}

func (ds *developmentServer) render(w http.ResponseWriter, req *http.Request) {
	err := ds.reload()
	if err != nil {
		logging.Log.Error(err)
	}
	category, _ := mux.Vars(req)["category"]
	p, _ := mux.Vars(req)["path"]
	if entry, ok := ds.entries[category+"/"+p]; ok {
		tpl, err := ds.pongo.FromFile(entry.TemplatePath)
		if err != nil {
			logging.Log.WithField("input", entry.TemplatePath).Error(ErrorTemplateLoader)
			logging.Log.Error(err)
			return
		}
		result := &client.QueryResult{}

		params := map[string]interface{}{}
		for k, v := range req.URL.Query() {
			params[k] = v[0]
		}

		if entry.ResolvedQuery != nil {
			logging.Log.WithField("input", *entry.QueryPath).Debug("running graphQL query")
			result, err = ds.client.Query(*entry.ResolvedQuery, params)
			if err != nil {
				logging.Log.Error(err)
				return
			}
			if len(result.Errors) > 0 {
				logging.Log.Error("got errors running the graphQL query")
				for k := range result.Errors {
					if len(result.Errors[k].Path) > 0 {
						logging.Log.WithField("path", strings.Join(result.Errors[k].Path, ".")).Error(result.Errors[k].Message)
					} else {
						logging.Log.Error(result.Errors[k].Message)
					}
				}
				return
			}
		}
		templateData := map[string]interface{}{}
		templateData["build"] = map[string]interface{}{
			"id":            123,
			"key":           uuid.NewString(),
			"repository_id": 123,
			"git_reference": map[string]interface{}{
				"type": "branch",
				"tag":  nil,
				"branch": map[string]interface{}{
					"name":       "main",
					"commit_sha": "123ABC",
				},
			},
		}
		templateData["vars"] = entry.Vars
		templateData["data"] = result.Data
		templateData["config"] = map[string]string{
			"asset_path": "http://" + ds.server.Addr + "/assets",
		}

		if v, ok := req.URL.Query()["dump"]; ok {
			logging.Log.WithField("input", *entry.QueryPath).Debug("dumping data for query")
			if v[0] == "1" {
				jsonData, err := json.Marshal(templateData)
				if err != nil {
					logging.Log.Error(err)
					return
				}
				w.Header().Set("content-type", "application/json")
				w.Write(jsonData)
				return
			}
		}

		logging.Log.WithField("input", entry.TemplatePath).Debug("rendering template entry")

		w.Header().Set("content-type", "text/html")
		err = tpl.ExecuteWriter(templateData, w)
		if err != nil {
			logging.Log.Error(err)
			return
		}
	} else {
		logging.Log.WithField("entry", p).Error("could not find entrypoint")
		w.WriteHeader(404)
	}
}

func (ds *developmentServer) index(w http.ResponseWriter, req *http.Request) {
	err := ds.reload()
	if err != nil {
		logging.Log.Error(err)
	}

	index, err := template.New("items").Parse("<html>\n<body>\n{{ range .Entries }}\n<li><a href=\"/render/{{ .Category }}/{{ .Path }}\">/{{ .Category }}/{{ .Path }}</a></li>\n{{ end }}\n</body>\n</html>")
	if err != nil {
		logging.Log.Error(err)
		return
	}

	if err := index.Execute(w, map[string]interface{}{
		"Entries": ds.entries,
	}); err != nil {
		logging.Log.Error(err)
		return
	}

}

func (ds *developmentServer) serveAsset(w http.ResponseWriter, req *http.Request) {
	file, _ := mux.Vars(req)["file"]

	filePath := filepath.Dir(ds.configFilePath) + "/dist/" + file
	if _, err := os.Stat(filePath); err == nil {
		fileData, err := ioutil.ReadFile(filePath)
		if err != nil {
			logging.Log.Error(err)
		}
		_, _ = w.Write(fileData)
		return

	} else if os.IsNotExist(err) {
		w.WriteHeader(404)
	} else {
		logging.Log.Error(err)
	}
}

func (ds *developmentServer) reload() error {
	configData, err := ioutil.ReadFile(ds.configFilePath)
	if err != nil {
		return err
	}
	definition := data.TemplateDefinition{}
	err = yaml.Unmarshal(configData, &definition)
	if err != nil {
		return err
	}

	entries := map[string]data.Route{}

	for _, entry := range definition.Config {
		routeVars := map[string]interface{}{}
		for _, v := range entry.Vars {
			routeVars[v.Name] = v.Value
		}
		for _, route := range entry.Routes {
			if route.QueryPath != nil {
				queryData, err := ioutil.ReadFile(filepath.Dir(ds.configFilePath) + "/" + *route.QueryPath)
				if err == nil {
					rq := string(queryData)
					route.Category = entry.Category
					route.ResolvedQuery = &rq
				}
			}
			route.Vars = routeVars
			entries[entry.Category+"/"+route.Path] = route
		}
	}

	ds.entries = entries
	ds.definition = &definition
	return nil
}
