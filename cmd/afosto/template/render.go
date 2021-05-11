package template

import (
	"fmt"
	"github.com/afosto/cli/pkg/auth"
	"github.com/afosto/cli/pkg/cli"
	"github.com/afosto/cli/pkg/logging"
	"github.com/afosto/cli/pkg/render"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"os"
	"strconv"
)

func GetCommands() []*cobra.Command {
	renderCmd := &cobra.Command{
		Use:   "render",
		Short: "Render template",
		Long:  `Start the local preview service`,
		Run: func(cmd *cobra.Command, args []string) {
			Render(cmd, args)
		}}

	renderCmd.Flags().IntP("port", "p", 3001, "Define port number for local proxy")
	renderCmd.Flags().StringP("file", "f", "", "Define path to the config file")

	return []*cobra.Command{renderCmd}
}

func Render(cmd *cobra.Command, args []string) {
	user := auth.GetUser()

	if user == nil {
		user = auth.LoadFromStorage()
	}

	if user == nil {
		user = auth.GetImplicitUser([]string{
			"openid",
			"email",
			"profile",
			"cnt:index:read",
			"iam:users:read",
			"iam:roles:read",
			"iam:tenants:read",
			"lcs:locations:read",
			"lcs:handling:read",
			"lcs:shipments:read",
			"odr:orders:read",
			"odr:coupons:read",
			"odr:invoices:read",
			"rel:contacts:read",
			"rel:identity:read",
		})
	}
	port, err := cmd.Flags().GetInt("port")
	if err != nil {
		logging.Log.Fatal("invalid port number given")
	}

	file, err := cmd.Flags().GetString("file")
	if err != nil {
		logging.Log.Fatal("invalid port number given")
	}
	if file == "" {
		cwd, err := os.Getwd()
		if err != nil {
			logging.Log.Fatal(err)
		}
		file = fmt.Sprintf("%s/%s", cwd, render.AfostoConfigFile)
	}

	server, err := render.GetDevelopmentServer("localhost", port, file, user)
	if err != nil {
		logging.Log.Fatal(err)
	}

	go server.Start()
	logging.Log.Debugf("✔ Started local preview proxy at %s:%d", "localhost", port)

	err = browser.OpenURL("http://localhost:" + strconv.Itoa(port) + "/index")
	if err != nil {
		logging.Log.Fatal("could not call browser")
	}
	cli.CloseHandler()

}
