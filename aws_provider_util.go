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
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/google/uuid"
	"os"
)

func dumpPrivateKeyPem(name string, key string) {
	f, err := os.Create(name)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write([]byte(key)); err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod(name, 0400); err != nil {
		log.Fatal(err)
	}
}

func DescribeInstances(svc *ec2.EC2, input *ec2.DescribeInstancesInput) *ec2.DescribeInstancesOutput {
	result, err := svc.DescribeInstances(input)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

func imageForRegion(region string) string {
	img, ok := imageByRegion[region]
	if !ok {
		log.Fatalf("no default image for region: %s", region)
	}
	return img
}

func CreateSpec(region string, nodes int, instanceType string) ClusterSpec {
	img := imageForRegion(region)
	spec := ClusterSpec{
		Region:    region,
		Instances: nil,
	}
	for i := 0; i < nodes; i++ {
		spec.Instances = append(spec.Instances, InstanceSpec{
			Name:  uuid.New().String(),
			Role:  RoleGenerator,
			Image: img,
			Type:  instanceType,
		})
	}
	spec.Instances = append(spec.Instances, InstanceSpec{
		Name:  uuid.New().String(),
		Role:  RoleMetrics,
		Image: img,
		Type:  DefaultMetricsInstanceType,
	})
	return spec
}
