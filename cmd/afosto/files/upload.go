package files

import (
	"github.com/afosto/cli/pkg/auth"
	"github.com/afosto/cli/pkg/client"
	"github.com/afosto/cli/pkg/logging"
	"github.com/gen2brain/dlgs"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

var _ io.Reader = (*os.File)(nil)

func GetCommands() []*cobra.Command {
	uploadCmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload files",

		Long: `Upload files to Afosto file storage`,
		Run: func(cmd *cobra.Command, args []string) {
			upload(cmd, args)
		}}

	uploadCmd.Flags().StringP("source", "s", "", "")
	uploadCmd.Flags().StringP("destination", "d", "", "")

	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Download files",

		Long: `Download files from Afosto file storage`,
		Run: func(cmd *cobra.Command, args []string) {
			download(cmd, args)
		}}

	downloadCmd.Flags().StringP("source", "s", "", "")
	downloadCmd.Flags().StringP("destination", "d", "", "")

	return []*cobra.Command{uploadCmd, downloadCmd}
}

func upload(cmd *cobra.Command, args []string) {
	user := auth.GetUser()
	if user == nil {
		user = auth.GetImplicitUser([]string{
			"openid",
			"email",
			"profile",
			"cnt:files:read",
			"cnt:files:write",
		})
	}

	ac := client.GetClient(user.TenantID, user.GetAccessToken())

	source, err := cmd.Flags().GetString("source")
	if err != nil {
		log.Fatal(err)
	}
	if source == "" {
		selectedSource, ok, err := dlgs.File("Select directory to upload", "", true)
		if err != nil {
			log.Fatal(err)
		}
		if !ok {
			log.Fatal("failed to select a directory")
		}
		source = selectedSource
	}

	destination, err := cmd.Flags().GetString("destination")
	if err != nil {
		log.Fatal(err)
	}
	if destination == "" {
		enteredDestination, ok, err := dlgs.Entry("enter the path to upload", "enter the directory to upload to", "/uploads/")
		if err != nil {
			log.Fatal(err)
		}
		if !ok {
			log.Fatal("failed to select a directory")
		}
		destination = enteredDestination
	}

	queue := make(chan string, 25)
	uploader := sync.WaitGroup{}
	matcher := regexp.MustCompile(".*(jpe?g|png|svg|css|csv|js|txt|doc|eot|json|xls|xlsx|pdf|xml|mp4|mov|zip|md)$")
	uploader.Add(1)
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for path := range queue {

				trimmedPath := path
				if idx := strings.LastIndex(trimmedPath, "/"); idx != -1 {
					trimmedPath = trimmedPath[0 : idx+1]
				}

				relativePath := strings.TrimLeft(trimmedPath, source)
				replacer := strings.NewReplacer("//", "/")
				destinationDir := replacer.Replace(strings.Join([]string{destination, relativePath}, "/"))

				signature, err := ac.GetSignature(destinationDir, "upsert")
				if err != nil {
					logging.Log.Warnf("✗ failed to get a signature url for  `%s`", destinationDir)
					return
				}

				file, err := ac.Upload(path, filepath.Base(path), signature)
				if err != nil {
					logging.Log.Errorf("✗ failed to upload  `%s`", path)
				} else {
					logging.Log.Infof("✔ Uploaded `%s` on url `%s`", file.Filename, file.Url)
				}
				uploader.Done()
			}

		}()
	}

	err = filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}
			if !matcher.MatchString(path) {
				logging.Log.Warnf("✗ invalid suffix for `%s`", path)
			}

			uploader.Add(1)
			queue <- path
			logging.Log.Infof("✔ added to queue `%s` ", path)

			return nil
		})

	if err != nil {
		log.Fatal(err)
	}

	uploader.Wait()
	logging.Log.Info("✔ Finished uploading all files")
}
