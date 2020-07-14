/*
 *    Copyright [2020] Sergey Kudasov
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package loadgen

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
)

var (
	orgId             = 1
	timerangeTemplate = "Grafana test data: %s/dashboard/db/observer?orgId=%d&from=%d&to=%d"
)

func TimerangeUrl(fromEpoch int64, toEpoch int64) {
	url := viper.GetString("grafana.url")
	log.Infof(timerangeTemplate, url, orgId, fromEpoch, toEpoch)
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func HumanReadableTestInterval(from string, to string) {
	log.Infof("Test time: %s - %s", from, to)
}

type UploadInput struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	PluginID string `json:"pluginId"`
	Value    string `json:"value"`
}

type ImportPayload struct {
	Dashboard Dashboard     `json:"dashboard"`
	Overwrite bool          `json:"overwrite"`
	Inputs    []UploadInput `json:"inputs"`
}

func uploadDashboard(login string, passwd string, url string, dashboard Dashboard) {
	payload := ImportPayload{
		Dashboard: dashboard,
		Overwrite: true,
		Inputs: []UploadInput{
			{
				Name:     "DS_LOCAL_GRAPHITE",
				Type:     "datasource",
				PluginID: "graphite",
				Value:    "Local Graphite",
			},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
	}
	dataBuf := bytes.NewBuffer(data)
	req, _ := http.NewRequest("POST", url, dataBuf)
	req.Header.Add(
		"Authorization",
		"Basic "+basicAuth(login, passwd))
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if resp.Body == nil {
		log.Fatalf("no upload body received, exiting")
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	log.Infof("import result: %s", respBody)
}

func UploadGrafanaDashboard() {
	title := viper.GetString("graphite.loadGeneratorPrefix")
	url := viper.GetString("grafana.url") + "/api/dashboards/import"
	login := viper.GetString("grafana.login")
	passwd := viper.GetString("grafana.password")
	log.Infof("importing grafana dashboard to %s", url)
	summaryDashboard := GrafanaGeneratorsSummaryDashboard(fmt.Sprintf("%s-summary", title), CollectYamlLabels())
	uploadDashboard(login, passwd, url, summaryDashboard)
	nodeDashboard := GrafanaGeneratorNodeDashboard(title, CollectYamlLabels(), title)
	uploadDashboard(login, passwd, url, nodeDashboard)
}
