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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"time"
)

const (
	DefaultMetricsInstanceType = "t2.micro"
	DefaultGrafanaPort         = "8181"
	DefaultVMRootUser          = "ec2-user"

	MetricsContainerCommand = "docker run -d -p 8181:80 -p 8125:8125/udp -p 8126:8126 --publish=2003:2003 --name kamon-grafana-dashboard kamon/grafana_graphite"
)

const (
	RoleMetrics   = "metrics"
	RoleGenerator = "generator"
)

type InfrastructureProviderAWS struct {
	client           *ec2.EC2
	session          *session.Session
	ClusterSpec      ClusterSpec
	RunningInstances map[string]*RunningInstance
}

type RunningInstance struct {
	Id              string
	Name            string
	KeyFileName     string
	Role            string
	PrivateKeyPem   string
	PublicDNSName   string
	PublicIPAddress string
}

type ClusterSpec struct {
	Region    string
	Instances []InstanceSpec
}

type InstanceSpec struct {
	Role  string
	Name  string
	Image string
	Type  string
}

var imageByRegion = map[string]string{
	// ec2 ECS optimised aws linux
	// TODO: find ecs optimized images with docker for all regions
	"us-east-2": "ami-0c0415cdff14e2a4a",
}

func stopInstance(svc *ec2.EC2, id string) {
	input := &ec2.StopInstancesInput{
		InstanceIds: []*string{
			aws.String(id),
		},
		DryRun: aws.Bool(true),
	}
	result, err := svc.StopInstances(input)
	awsErr, ok := err.(awserr.Error)
	if ok && awsErr.Code() == "DryRunOperation" {
		input.DryRun = aws.Bool(false)
		result, err = svc.StopInstances(input)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Infof("instance stopped: %s", result.StoppingInstances)
		}
	} else {
		log.Fatal(err)
	}
}

func (m *InfrastructureProviderAWS) createInstances() {
	for _, instanceSpec := range m.ClusterSpec.Instances {
		privateKey := createKeyPair(m.client, instanceSpec.Name)
		runResult, err := m.client.RunInstances(&ec2.RunInstancesInput{
			ImageId:      aws.String(instanceSpec.Image),
			InstanceType: aws.String(instanceSpec.Type),
			MinCount:     aws.Int64(1),
			MaxCount:     aws.Int64(1),
			KeyName:      aws.String(instanceSpec.Name),
			// must be security group with tcp 22, 8181 allowance, configured once for aws account
			//SecurityGroupIds: []*string{aws.String("sg-6dc1de0b")},
		})
		if err != nil {
			log.Fatal(err)
		}
		if len(runResult.Instances) == 0 || len(runResult.Instances) > 1 {
			log.Fatalf("failed to start ec2 instance: %s", err)
		}
		id := runResult.Instances[0].InstanceId
		log.Infof("instance created: %s (aws_id: %s)", instanceSpec.Name, *id)
		_, errtag := m.client.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{runResult.Instances[0].InstanceId},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(instanceSpec.Name),
				},
				{
					Key:   aws.String("Id"),
					Value: id,
				},
			},
		})
		if errtag != nil {
			log.Errorf("Could not create tags for instance", id, errtag)
		}
		log.Infof("instance tagged: %s", *id)
		m.RunningInstances[instanceSpec.Name] = &RunningInstance{
			*id,
			instanceSpec.Name,
			"",
			instanceSpec.Role,
			privateKey,
			// must be acquired later, when vm in state "running"
			"",
			"",
		}
	}
}

func (m *InfrastructureProviderAWS) collectPublicAddresses() {
	for _, r := range m.RunningInstances {
		res := DescribeInstances(m.client, filterById(r.Id))
		r.PublicIPAddress = *res.Reservations[0].Instances[0].PublicIpAddress
		r.PublicDNSName = *res.Reservations[0].Instances[0].PublicDnsName
		log.Debugf("addresses assigned: %s (%s)", r.PublicDNSName, r.PublicIPAddress)
	}
}

func (m *InfrastructureProviderAWS) provision() {
	for _, r := range m.RunningInstances {
		userDnsString := fmt.Sprintf("%s@%s", DefaultVMRootUser, r.PublicDNSName)
		switch r.Role {
		case RoleMetrics:
			log.Infof("provisioning metrics collector vm: %s", r.Name)
			m.Exec(r.Name, MetricsContainerCommand)
			log.Infof("grafana deployed: %s", grafanaUrl(r.PublicDNSName))
		case RoleGenerator:
			log.Infof("generator deployed")
			loadScriptDir := viper.GetString("load_scripts_dir")
			remoteRoot := fmt.Sprintf("%s:/home/%s", userDnsString, DefaultVMRootUser)
			UploadSuiteCommand(loadScriptDir, remoteRoot, r.KeyFileName)
		default:
			log.Fatalf("unknown vm role: %s", r.Role)
		}
		log.Infof("connection string: \nssh -i %s %s", userDnsString, userDnsString)
	}
}

func (m *InfrastructureProviderAWS) dumpPrivateKeys() {
	for _, r := range m.RunningInstances {
		r.KeyFileName = fmt.Sprintf("%s@%s", DefaultVMRootUser, r.PublicDNSName)
		dumpPrivateKeyPem(r.KeyFileName, r.PrivateKeyPem)
	}
}

// Bootstrap creates vms according to spec and wait until all vm in state "running"
func (m *InfrastructureProviderAWS) Bootstrap() {
	m.createInstances()
	for i := 0; i < 50; i++ {
		time.Sleep(5 * time.Second)
		if m.assureRunning() {
			log.Info("all instances are running")
			break
		}
	}
	m.collectPublicAddresses()
	m.dumpPrivateKeys()
	// TODO: general retry function, even if VM is in state "running" ssh may be unavailable,
	//  has no flag to know it for sure
	log.Infof("awaiting ssh is available")
	time.Sleep(60 * time.Second)
	m.provision()
}

func filterById(id string) *ec2.DescribeInstancesInput {
	return &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Id"),
				Values: []*string{aws.String(id)},
			},
		},
	}
}

func (m *InfrastructureProviderAWS) assureRunning() bool {
	values := make([]*string, 0)
	for _, i := range m.RunningInstances {
		values = append(values, aws.String(i.Id))
	}
	filter := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Id"),
				Values: values,
			},
		},
	}
	res := DescribeInstances(m.client, filter)
	log.Debugf("describe response: %s\n", res)
	for _, r := range res.Reservations {
		if *r.Instances[0].State.Name != "running" {
			return false
		}
	}
	return true
}

func (m *InfrastructureProviderAWS) Exec(vmName string, cmd string) {
	for _, r := range m.RunningInstances {
		if r.Name == vmName {
			client, sess, err := connectSSHToHost(DefaultVMRootUser, r.PublicIPAddress, r.PrivateKeyPem)
			if err != nil {
				panic(err)
			}
			log.Debugf("executing cmd on vm %s: %s", r.Id, cmd)
			out, err := sess.CombinedOutput(cmd)
			if err != nil {
				log.Fatal(err)
			}
			log.Debugf(string(out))
			client.Close()
		}
	}
}

func newEC2Session(region string) (*ec2.EC2, *session.Session) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		log.Fatal(err)
	}
	return ec2.New(sess), sess
}

func createKeyPair(svc *ec2.EC2, name string) string {
	result, err := svc.CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: aws.String(name),
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("Created key pair %q %s\n%s\n",
		*result.KeyName, *result.KeyFingerprint,
		*result.KeyMaterial)
	return *result.KeyMaterial
}

func NewInfrastructureProviderAWS(spec ClusterSpec) *InfrastructureProviderAWS {
	client, sess := newEC2Session(spec.Region)
	return &InfrastructureProviderAWS{
		session:          sess,
		client:           client,
		ClusterSpec:      spec,
		RunningInstances: make(map[string]*RunningInstance, 0),
	}
}

func grafanaUrl(name string) string {
	return fmt.Sprintf("http://%s:%s", name, DefaultGrafanaPort)
}

func connectSSHToHost(user, ip, privateKeyPem string) (*ssh.Client, *ssh.Session, error) {
	signer, err := ssh.ParsePrivateKey([]byte(privateKeyPem))
	if err != nil {
		log.Fatal(err)
	}

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
	}
	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	client, err := ssh.Dial("tcp", ip+":22", sshConfig)
	if err != nil {
		return nil, nil, err
	}

	sess, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, sess, nil
}
