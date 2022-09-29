package files

import "github.com/spf13/cobra"

func GetCommands() []*cobra.Command {
	uploadCmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload files",
		Long:  `Upload files to Afosto file storage`,
		Run: func(cmd *cobra.Command, args []string) {
			upload(cmd, args)
		}}

	uploadCmd.Flags().StringP("source", "s", "", "Select the source file or directory")
	uploadCmd.Flags().StringP("destination", "d", "", "Choose a path to upload the sources file(s) into")
	uploadCmd.Flags().BoolP("private", "p", false, "Whether the uploaded files should be private")

	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Download files",
		Long:  `Download files from Afosto file storage`,
		Run: func(cmd *cobra.Command, args []string) {
			download(cmd, args)
		}}

	downloadCmd.Flags().StringP("source", "s", "", "Select the source file or directory")
	downloadCmd.Flags().StringP("destination", "d", "", "Choose a path to download the sources file(s) into")

	return []*cobra.Command{uploadCmd, downloadCmd}
}
