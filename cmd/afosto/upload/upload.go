package upload

import (
	"context"
	"github.com/afosto/cli/pkg/auth"
	"github.com/afosto/cli/pkg/client"
	"github.com/afosto/cli/pkg/logging"
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

	dir, _ := os.Getwd()

	renderCmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload template",

		Long: `Start the local preview service`,
		Run: func(cmd *cobra.Command, args []string) {
			upload(cmd, args)
		}}

	renderCmd.Flags().StringP("source", "s", dir, "")
	renderCmd.Flags().StringP("destination", "d", "/uploads/", "")

	return []*cobra.Command{renderCmd}
}

func upload(cmd *cobra.Command, args []string) {
	source, err := cmd.Flags().GetString("source")

	if err != nil {
		log.Fatal(err)
	}

	destination, err := cmd.Flags().GetString("destination")

	if err != nil {
		log.Fatal(err)
	}

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
	ctx := context.WithValue(context.Background(), client.Jwt, user.GetAccessToken())

	fileClient := client.NewFileClient()
	queue := make(chan string, 25)
	uploader := sync.WaitGroup{}
	matcher := regexp.MustCompile(".*(jpe?g|png|svg|css|csv|js|txt|doc|eot|json|xls|xlsx|pdf|xml|mp4|mov|zip|md)$")
	uploader.Add(1)
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for path := range queue {
				file1, err := os.Open(path)
				if err != nil {
					logging.Log.Warnf("✗ could not open `%s`", path)
				}

				filePath := path

				if idx := strings.LastIndex(filePath, "/"); idx != -1 {
					filePath = filePath[0 : idx+1]
				}

				relativePath := strings.TrimLeft(filePath, source)

				replacer := strings.NewReplacer("//", "/")

				destinationDir := replacer.Replace(strings.Join([]string{destination, relativePath}, "/"))

				//
				finfo, err := file1.Stat()
				if err != nil {
					logging.Log.Warnf("✗ could not get stats of `%s`", path)
				}

				signature, err := fileClient.GetPublicSignature(ctx, destinationDir, "upsert")
				if err != nil {
					logging.Log.Warnf("✗ failed to get a signature url for  `%s`", destinationDir)
				}

				file, err := fileClient.Upload(ctx, file1, finfo.Name(), signature)
				if err != nil {
					logging.Log.Errorf("✗ failed to upload  `%s`", path)
				} else {
					logging.Log.Infof("✔ Uploaded `%s` on url `%s`", file.Filename, file.Url)
				}
				uploader.Done()
				file1.Close()
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

			//fmt.Println(path)

			return nil
		})

	if err != nil {
		log.Fatal(err)
	}

	uploader.Wait()

	logging.Log.Info("✔ Finished uploading all files")
}
