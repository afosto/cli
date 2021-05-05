package files

import (
	"github.com/afosto/cli/pkg/auth"
	"github.com/afosto/cli/pkg/client"
	"github.com/afosto/cli/pkg/logging"
	"github.com/gen2brain/dlgs"
	"github.com/spf13/cobra"
	"os"
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

	if _, err := os.Stat(destination); os.IsExist(err) {
		err := os.Mkdir(destination, 0755)
		if err != nil {
			logging.Log.Fatal("Destination path does not yet exist and could not create it")
		}
	}

	files, err := ac.ListDirectory(source)
	if err != nil {
		logging.Log.Fatal(err)
	}

	logging.Log.Info(files)

}
