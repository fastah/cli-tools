/*
Copyright © 2020 Copyright © 2020 Blackbuck Computing Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Prepares the tool for the further use by saving the specified Fastah API key in a local config file",
	Long:  `Provide authentication information to the tool so that authorization credentials are stored in $HOME/.whereis.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		//fmt.Printf("init called with Fastah API key %v\n", cmd.Flag("fastah-api-key").Value)
		saveAPIKeyToConfigFile(cmd.Flag("fastah-api-key").Value.String())
	},
}

func saveAPIKeyToConfigFile(key string) {
	_ = viper.AllSettings()
	viper.SetDefault("fastah-api-key", key)
	viper.Set("fastah-api-key", key)
	viper.WriteConfig()
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	initCmd.PersistentFlags().String("fastah-api-key", "", "Fastah API Key from console.api.getfastah.com")
	initCmd.MarkPersistentFlagRequired("fastah-api-key")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
