package files

import (
	"github.com/afosto/cli/pkg/auth"
	"github.com/afosto/cli/pkg/client"
	"github.com/afosto/cli/pkg/data"
	"github.com/afosto/cli/pkg/logging"
	"github.com/gen2brain/dlgs"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"sync"
)

func download(cmd *cobra.Command, args []string) {
	user := auth.GetUser()

	if user == nil {
		user = auth.LoadFromStorage()
	}

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
	source = strings.Trim(source, "/")

	destination, err := cmd.Flags().GetString("destination")

	if destination == "" {
		selectedDestination, ok, err := dlgs.File("Select the download directory", "", true)
		if err != nil {
			logging.Log.Fatal(err)
		}
		if !ok {
			logging.Log.Fatal("Failed to select a directory")
		}
		destination = selectedDestination
	}

	destination = strings.TrimRight(destination, "/")

	downloadQueue := make(chan data.File, 10)

	var wg sync.WaitGroup

	go downloadHandler(downloadQueue, ac, source, destination, &wg)
	logging.Log.Infof("✔ Started listing Directories`")
	directories, err := ac.ListDirectories(source)
	if err != nil {
		logging.Log.Fatal(err)
	}
	logging.Log.Infof("✔ Finished listing directories`")

	for _, directory := range directories {
		var files []data.File
		cursor := ""
		for {
			files, cursor, err = ac.ListDirectory(strings.TrimRight(directory, "/"), cursor)
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
	}

	wg.Wait()

	logging.Log.Infof("✔ Downloaded all files from `%s` to `%s`", source, destination)

}

func downloadHandler(downloadQueue <-chan data.File, ac *client.AfostoClient, source string, destination string, wg *sync.WaitGroup) {
	for file := range downloadQueue {
		go func(file data.File, source string, destination string, wg *sync.WaitGroup) {
			defer wg.Done()
			fileUri, err := url.Parse(file.Url)

			if err != nil {
				logging.Log.Error(err)
				return
			}
			trimmedPath := source
			if idx := strings.LastIndex(trimmedPath, "/"); idx != -1 {
				trimmedPath = trimmedPath[0 : idx+1]
			}

			destinationDir := destination + strings.TrimLeft(file.Dir, source)

			b, err := ac.Download(fileUri)
			if err != nil {
				logging.Log.Error(err)
				return
			}

			if err := os.MkdirAll(destinationDir, 0755); err != nil {
				logging.Log.Error("✗ Destination path does not yet exist and could not create it")
				return

			}

			if err := ioutil.WriteFile(destinationDir+"/"+file.Filename, b, 0664); err != nil {
				logging.Log.Error(err)
			}
			logging.Log.Infof("✔ Downloaded `%s` on from `%s`", file.Filename, file.Url)

		}(file, source, destination, wg)

	}
}
