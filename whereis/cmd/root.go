/*
Copyright © 2020 Blackbuck Computing Inc.

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
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/oschwald/geoip2-golang"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "whereis",
	Short: "Finds approximate location, city, country or timezone for a specified IP address.",
	Long: `Finds approximate location, city, country or timezone for a specified IP address. For example:

whereis 8.8.4.4`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		//fmt.Printf("In root command worker function (arg count = %d, flag count = %d)\n", len(args), cmd.Flags().NFlag())
		ipString, _ := cmd.Flags().GetString("ip")
		//fmt.Printf("In root command worker function (flag = %s)\n", ipString)

		if ipString == "-" {
			apiKey := viper.GetString("fastah-api-key")
			processFromStdin(apiKey, true)
		} else {
			ip, err := cmd.Flags().GetString("ip")
			if err != nil {
				fmt.Printf("Expected a valid IP in the --ip argument ( %v )\n", err)
				os.Exit(1)
			}
			fmt.Printf("TODO: Implement single IP lookup for %s\n", ip)
		}
	},
}

func processFromStdin(fastahAPIkey string, comparedWithMMDB bool) {

	var mmdbReader *geoip2.Reader

	if comparedWithMMDB {
		// Find home directory to infer MMDB path
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		mmdbReader, err = geoip2.Open(home + "/GeoLite2-City.mmdb")
		if err != nil || mmdbReader == nil {
			log.Fatal(err)
		}
	}

	// Secure TLS settings to minimize MITM mischief
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		MaxIdleConnsPerHost: http.DefaultMaxIdleConnsPerHost * 5,
		TLSHandshakeTimeout: time.Second * 5,
		IdleConnTimeout:     0,
		ForceAttemptHTTP2:   true,
	}
	// http2.ConfigureTransport(tr)
	httpclient := &http.Client{Transport: tr}

	fastahEndPointBase := "https://ep.api.getfastah.com/whereis/v1/json/"

	// Output formatting tool
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"IP", "Country (F)", "Country (M)", "City (F)", "City (M)", "Lat/Lng (F)", "Lat/Lng (M)", "TZ (F)", "TZ (M)"})

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		ipRaw := strings.TrimSpace(scanner.Text())
		if ipRaw == "" {
			continue
		}
		ip := net.ParseIP(ipRaw)
		if ip == nil {
			continue
		}

		// Prep result printing apparatus
		resultRow := make([]string, 9, 9)
		resultRow[0] = ip.String()

		mmdbRec, err := mmdbReader.City(ip)
		if err != nil {
			log.Fatal(err)
		}

		resultRow[2] = mmdbRec.Country.IsoCode
		resultRow[4] = mmdbRec.City.Names["en"]
		resultRow[6] = fmt.Sprintf("%0.2f, %0.2f", mmdbRec.Location.Latitude, mmdbRec.Location.Longitude)
		resultRow[8] = mmdbRec.Location.TimeZone

		url := fastahEndPointBase + ip.String()
		req, err := http.NewRequest("GET", url, nil)
		req.Header.Set("Fastah-Key", fastahAPIkey)
		resp, err := httpclient.Do(req)
		if err != nil {
			log.Panicln(err)
		}
		defer resp.Body.Close()

		// Parse the success or fail output of the Fastah API
		if resp.StatusCode == 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Panicln(err)
			}
			var ipInfo LocationResponse
			err = json.Unmarshal(body, &ipInfo)
			if err != nil {
				fmt.Println("error:", err)
			}
			resultRow[1] = *ipInfo.LocationData.CountryCode
			resultRow[3] = *ipInfo.LocationData.CityName
			resultRow[5] = fmt.Sprintf("%0.2f, %0.2f", *ipInfo.LocationData.Lat, *ipInfo.LocationData.Lng)
			resultRow[7] = *ipInfo.LocationData.Tz
			table.Append(resultRow)
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Panicln(err)
			}
			var errorPayload ProblemResponseError
			err = json.Unmarshal(body, &errorPayload)
			if err != nil {
				fmt.Printf("problem parsing error message body, status code = %d, err = %v", resp.StatusCode, err)
			} else {
				fmt.Printf("Problem with Fastah API call; HTTP code %d ( %s )\n", resp.StatusCode, errorPayload.Message)
			}
			resultRow[1], resultRow[3], resultRow[5], resultRow[7] = "💩", "💩", "💩", "💩"
		}
		table.Append(resultRow)
	}

	if scanner.Err() != nil {
		log.Panicln(scanner.Err())
	}

	if comparedWithMMDB && mmdbReader != nil {
		mmdbReader.Close()
	}

	table.Render()
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.whereis.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringP("ip", "i", "-", "IP address to lookup")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".whereis.yaml" (with extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".whereis.yaml")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		//fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
	// Example below of how to read any other key already in the config file
	// fmt.Printf("Value of key GREETING is %s\n", viper.GetString("greeting"))
}

// LocationResponse and children are copied as-is from the go-swagger generated model from the OpenAPI spec file
type LocationResponse struct {

	// location data
	LocationData *LocationResponseLocationData `json:"locationData,omitempty"`
}

type LocationResponseLocationData struct {

	// City name
	// Required: true
	CityName *string `json:"cityName"`

	// 2-letter continent code
	// Required: true
	ContinentCode *string `json:"continentCode"`

	// Country name as an ISO 3166-1 code
	// Required: true
	CountryCode *string `json:"countryCode"`

	// Country name as a display string
	// Required: true
	CountryName *string `json:"countryName"`

	// Latitude representing the approximate location
	// Required: true
	Lat *float64 `json:"lat"`

	// Longitude representing the approximate location
	// Required: true
	Lng *float64 `json:"lng"`

	// Time zone for the region
	// Required: true
	Tz *string `json:"tz"`
}

type ProblemResponseError struct {

	// developer-friendly description of what went wrong
	Message string `json:"message,omitempty"`
}
