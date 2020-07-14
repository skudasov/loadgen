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
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	suiteBinaryName = "./load_suite"
	suiteMain       = "%s/cmd/load/main.go"
	scpNoHostCheck  = "scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
	scpCmd          = "scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i %s -pr %s %s"
)

func BuildSuiteCommand(testDir string, platform string) {
	if err := os.Setenv("GOOS", platform); err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", suiteBinaryName, fmt.Sprintf(suiteMain, testDir))
	res, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("failed to build suite: out:%s err: %s\n", res, err)
	}
}

func RunSuiteCommand(cfgPath string) {
	cmd := exec.Command(suiteBinaryName, "-config", cfgPath)
	res, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("failed to run suite: out:%s err: %s\n", res, err)
	}
}

func execOSCmd(cmdStr string) string {
	log.Debugf("executing cmd: %s", cmdStr)
	command := strings.Split(cmdStr, " ")
	cmd := exec.Command(command[0], command[1:]...)
	res, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("failed to upload suite: out:%s, err: %s", res, err)
	}
	return string(res)
}

func UploadSuiteCommand(testDir string, remoteRootDir string, keyPath string) {
	remotePath := path.Join(remoteRootDir, testDir)
	log.Infof("syncing test dir: %s to remote: %s", testDir, remotePath)
	cmd2Str := fmt.Sprintf(scpCmd, keyPath, testDir, remotePath)
	execOSCmd(cmd2Str)
	log.Infof("uploading suite binary to: %s", remotePath)
	cmdStr := fmt.Sprintf(scpCmd, keyPath, suiteBinaryName, remotePath)
	execOSCmd(cmdStr)
}

func GenerateNewTestCommand(testDir string, label string) {
	labels := CollectLabels(testDir)
	labels = append(labels, LabelKV{
		Label:     label,
		LabelName: NewLabelName(label),
	})
	CodegenAttackersFile(testDir, labels)
	CodegenChecksFile(testDir, labels)
	CodegenLabelsFile(testDir, labels)
	CodegenAttackerFile(testDir, label)
	GenerateSingleRunConfig(testDir, label)
}
