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
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

whereis --ip=202.94.72.116`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		//fmt.Printf("In root command worker function (arg count = %d, flag count = %d)\n", len(args), cmd.Flags().NFlag())
		ipString, _ := cmd.Flags().GetString("ip")
		//fmt.Printf("In root command worker function (flag = %s)\n", ipString)

		if ipString == "-" {
			apiKey := viper.GetString("fastah-api-key")
			processFromStdin(context.Background(), apiKey, true)
		} else {
			ip, err := cmd.Flags().GetString("ip")
			if err != nil {
				panic(fmt.Sprintf("Expected a valid IP in the --ip argument ( %v )\n", err))
			}
			fmt.Printf("TODO: Implement single IP lookup for %s\n", ip)
		}
	},
}

// processFromStdin processes multiple IP address read from os.Stdin, one line at a time.
func processFromStdin(ctx context.Context, fastahAPIkey string, compareMMDB bool) {

	// Set up high performance and secure HTTP/2 client configuration to communication with the Fastah API endpoint
	// Turns on HTTP/2 via ForceAttemptHTTP2=true - ensures that responses come quickly on a multiplex connection, with no head-of-line blocking.
	// Additionally, specifies minimum TLS version of 1.2, since it provides minimum-credible transport security.
	tr := &http.Transport{
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout: time.Second * 5,
		IdleConnTimeout:     0,
		ForceAttemptHTTP2:   true,
	}
	// http2.ConfigureTransport(tr)
	httpclient := &http.Client{Transport: tr}
	fastahEndPointBase := "https://ep.api.getfastah.com/whereis/v1/json/"

	// Only useful if we are comparing results with an MMDB database
	var mmdbReader *geoip2.Reader
	if compareMMDB {
		// Find home directory to infer MMDB path
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		mmdbReader, err = geoip2.Open(home + "/GeoLite2-City.mmdb")
		if err != nil {
			compareMMDB = false
			//fmt.Fprintf(os.Stderr, "Warning: disabling MMDB use (%v)\n", err)
		}
	}

	// Pretty printing of output results
	table := tablewriter.NewWriter(os.Stdout)
	tableHeader := []string{}
	if compareMMDB {
		tableHeader = []string{"IP", "Country (F)", "Country (M)", "City (F)", "City (M)", "Lat/Lng (F)", "Lat/Lng (M)", "TZ (F)", "TZ (M)"}
	} else {
		tableHeader = ([]string{"IP", "Country", "City", "Lat/Lng", "TZ"})
	}
	table.SetHeader(tableHeader)
	// Read stdin, one IP per line
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
		resultRow := make([]string, len(tableHeader), len(tableHeader))
		resultRow[0] = ip.String()

		if compareMMDB {
			mmdbRec, err := mmdbReader.City(ip)
			if err != nil {
				panic(fmt.Sprintf("Problem looking up IP (%s) in MMDB file: err = %v", ip.String(), err))
			}
			resultRow[2] = mmdbRec.Country.IsoCode
			resultRow[4] = mmdbRec.City.Names["en"]
			resultRow[6] = fmt.Sprintf("%0.2f, %0.2f", mmdbRec.Location.Latitude, mmdbRec.Location.Longitude)
			resultRow[8] = mmdbRec.Location.TimeZone
		}

		url := fastahEndPointBase + ip.String()
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		req.Header.Set("Fastah-Key", fastahAPIkey)
		resp, err := httpclient.Do(req)
		if err != nil {
			panic(fmt.Sprintf("Problem sending HTTP request to Fastah API: err = %v", err))
		}
		defer resp.Body.Close()

		// Parse the success or fail output of the Fastah API
		if resp.StatusCode == 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(fmt.Sprintf("Problem reading Fastah API OK response: err = %v", err))
			}
			var ipInfo LocationResponse
			err = json.Unmarshal(body, &ipInfo)
			if err != nil {
				panic(fmt.Sprintf("Problem parsing Fastah API OK response (data model changed?): err = %v", err))
			}
			skipCount := 1
			if compareMMDB {
				skipCount = 2
			}
			resultRow[1] = *ipInfo.LocationData.CountryCode
			resultRow[1+1*skipCount] = *ipInfo.LocationData.CityName
			resultRow[1+2*skipCount] = fmt.Sprintf("%0.2f, %0.2f", *ipInfo.LocationData.Lat, *ipInfo.LocationData.Lng)
			resultRow[1+3*skipCount] = *ipInfo.LocationData.Tz
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(fmt.Sprintf("Problem reading Fastah API response body: status code = %d, err = %v", resp.StatusCode, err))
			}
			var errorPayload ProblemResponseError
			err = json.Unmarshal(body, &errorPayload)
			if err != nil {
				panic(fmt.Sprintf("Problem parsing Fastah API error (data model changed?): status code = %d, err = %v", resp.StatusCode, err))
			} else {
				fmt.Fprintf(os.Stderr, "Problem with Fastah API call; HTTP code %d ( %s )\n", resp.StatusCode, errorPayload.Message)
			}
			resultRow[1], resultRow[3], resultRow[5], resultRow[7] = "💩", "💩", "💩", "💩"
		}
		table.Append(resultRow)
	}

	if scanner.Err() != nil {
		panic(scanner.Err())
	}

	if compareMMDB && mmdbReader != nil {
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
