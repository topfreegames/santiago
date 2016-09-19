// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var smokeURL string

//smokeCmd represents the version command
var smokeCmd = &cobra.Command{
	Use:   "smoke",
	Short: "runs smoke tests against santiago",
	Long:  `This command runs smoke tests against santiago. It uses https://requestb.in to inspect that Santiago actually requested the page.`,
	Run: func(cmd *cobra.Command, args []string) {
		doSmokeTest()
	},
}

func doSmokeTest() {
	requestBinName := getRequestBin()
	requestSantiagoHook(requestBinName)
	time.Sleep(1 * time.Second)
	requestCount := getRequestCount(requestBinName)
	if requestCount != 1 {
		binURL := fmt.Sprintf("https://requestb.in/%s?inspect", requestBinName)
		panic(fmt.Sprintf(
			"The request count is supposed to be exactly 1. It was %d. Please inspect the requests done at %s.",
			requestCount,
			binURL,
		))
	}
}

func getRequestBin() string {
	jsonStr := `{"private":false}`
	url := "https://requestb.in/api/v1/bins"
	fmt.Println("Creating requestb.in...")
	status, obj, err := doRequest("POST", url, jsonStr)
	if err != nil {
		panic(err)
	}
	if status != 200 {
		panic("Request to create requestb.in failed!")
	}

	fmt.Printf(
		"requestb.in created successfully. To view details about your bin, access https://requestb.in/%s?inspect\n",
		obj["name"].(string),
	)
	return obj["name"].(string)
}

func requestSantiagoHook(requestBinName string) {
	requestBinPostURL := fmt.Sprintf("http://requestb.in/%s", requestBinName)
	data := fmt.Sprintf(`{"ts":%d}`, time.Now().Unix())
	santiagoURL := fmt.Sprintf(`%s/hooks?method=POST&url=%s`, smokeURL, requestBinPostURL)
	fmt.Printf("Requesting a hook dispatch at %s...\n", santiagoURL)
	status, _, err := doRequest("POST", santiagoURL, data)
	if err != nil {
		panic(fmt.Sprintf("Could not request santiago hook: %s\n", err.Error()))
	}
	if status != 200 {
		panic(fmt.Sprintf("Could not request santiago hook! Status Code: %d\n", status))
	}
}

func getRequestCount(requestBinName string) int {
	fmt.Println("Getting requestb.in count...")
	status, obj, err := doRequest("GET", fmt.Sprintf("https://requestb.in/api/v1/bins/%s", requestBinName), "")

	if err != nil {
		panic(err)
	}
	if status != 200 {
		panic("Request to get requestb.in count failed!")
	}

	requestCount := int(obj["request_count"].(float64))
	fmt.Printf(
		"requestb.in request count retrieved successfully! %d requests made so far.\n",
		requestCount,
	)
	return requestCount
}

func doRequest(method, url, body string) (int, map[string]interface{}, error) {
	bodyObj := bytes.NewBuffer([]byte(body))
	req, err := http.NewRequest(method, url, bodyObj)
	if err != nil {
		return 500, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 500, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return resp.StatusCode, nil, err
	}

	rBody, _ := ioutil.ReadAll(resp.Body)
	if string(rBody) == "OK" {
		return resp.StatusCode, nil, nil
	}
	var obj map[string]interface{}
	err = json.Unmarshal(rBody, &obj)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, obj, nil
}

func init() {
	RootCmd.AddCommand(smokeCmd)
	smokeCmd.Flags().StringVarP(&smokeURL, "url", "u", "", "URL to run smoke test against")
}
