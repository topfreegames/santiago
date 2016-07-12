// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/topfreegames/santiago/metadata"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "returns Santiago version",
	Long:  `returns Santiago version`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Santiago v%s\n", metadata.VERSION)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
