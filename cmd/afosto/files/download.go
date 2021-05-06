package files

import (
	"bytes"
	"github.com/afosto/cli/pkg/auth"
	"github.com/afosto/cli/pkg/client"
	"github.com/afosto/cli/pkg/data"
	"github.com/afosto/cli/pkg/logging"
	"github.com/gen2brain/dlgs"
	"github.com/spf13/cobra"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func download(cmd *cobra.Command, args []string) {
	user := auth.GetUser()
	if user == nil {
		user = auth.GetImplicitUser([]string{
			"openid",
			"email",
			"profile",
			"cnt:files:read",
		})
	}

	ac := client.GetClient(user.TenantID, user.GetAccessToken())
	source, err := cmd.Flags().GetString("source")
	if err != nil {
		logging.Log.Fatal(err)
	}
	if source == "" {
		selectedSource, ok, err := dlgs.File("Select source path", "", true)
		if err != nil {
			logging.Log.Fatal(err)
		}
		if !ok {
			logging.Log.Fatal("failed to select a source path")
		}
		source = selectedSource
	}

	destination, err := cmd.Flags().GetString("destination")
	if err != nil {
		logging.Log.Fatal(err)
	}
	destination = strings.TrimRight(destination, "/")

	if _, err := os.Stat(destination); os.IsNotExist(err) {
		err := os.Mkdir(destination, 0755)
		if err != nil {
			logging.Log.Fatal("Destination path [" + destination + "] does not yet exist and could not create it")
		}
	}

	downloadQueue := make(chan data.File, 10)
	var wg sync.WaitGroup
	go downloadHandler(downloadQueue, ac, destination, &wg)

	cursor := ""
	var files []data.File
	for {
		files, cursor, err = ac.ListDirectory(source, cursor)
		if err != nil {
			logging.Log.Fatal(err)
		}
		for _, file := range files {
			wg.Add(1)
			downloadQueue <- file
		}
		if len(files) < 25 {
			break
		}
	}
	wg.Wait()

	logging.Log.Infof("✔ Downloaded all files from `%s` to `%s`", source, destination)

}

func downloadHandler(downloadQueue <-chan data.File, ac *client.AfostoClient, destination string, wg *sync.WaitGroup) {
	for file := range downloadQueue {
		go func(file data.File, wg *sync.WaitGroup) {
			u, err := url.Parse(file.Url)
			if err != nil {
				logging.Log.Error(err)

			}
			b, err := ac.Download(u)
			logging.Log.Infof("✔ Downloaded `%s` on from `%s`", file.Filename, file.Url)
			path := destination + "/" + file.Dir + "/" + file.Filename
			targetPath := filepath.Dir(path)
			if _, err := os.Stat(targetPath); os.IsNotExist(err) {
				err := os.Mkdir(targetPath, 0755)
				if err != nil {
					logging.Log.Error("Destination path does not yet exist and could not create it")

				}
			}
			out, err := os.Create(path)
			if err != nil {
				logging.Log.Error(err)
			}
			defer out.Close()
			_, err = io.Copy(out, bytes.NewReader(b))
			if err != nil {
				logging.Log.Error(err)
			}
			wg.Done()
		}(file, wg)

	}
}
