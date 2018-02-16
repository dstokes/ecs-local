package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/fsouza/go-dockerclient"
	flag "github.com/ogier/pflag"
)

const (
	exitCodeOk             int = 0
	exitCodeError          int = 1
	exitCodeDockerError    int = 2
	exitCodeFlagParseError     = 10 + iota
	exitCodeAWSError
)

var (
	Version = "No version specified"
)

var (
	f = flag.NewFlagSet("flags", flag.ContinueOnError)

	// options
	helpFlag    = f.BoolP("help", "h", false, "help")
	profileFlag = f.StringP("profile", "p", "", "AWS profile")
	regionFlag  = f.StringP("region", "r", "us-east-1", "AWS region")
	verboseFlag = f.BoolP("verbose", "v", false, "verbose")
	versionFlag = f.Bool("version", false, "version")
)

const helpString = `Usage:
  ecs-local [-hv] [--profile=aws_profile] [--region=aws_region]

Flags:
  -h, --help    Print this help message
  -p, --profile The AWS profile to use
  -r, --region  The AWS region the table is in
  -v, --verbose Verbose logging
      --version
`

func main() {
	if err := f.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(exitCodeFlagParseError)
	}
	args := f.Args()

	if *helpFlag == true {
		fmt.Print(helpString)
		os.Exit(exitCodeOk)
	}

	if *versionFlag == true {
		fmt.Printf("%s %s\n", filepath.Base(os.Args[0]), Version)
		os.Exit(exitCodeOk)
	}

	if len(args) < 1 {
		fmt.Print(helpString)
		os.Exit(exitCodeOk)
	}

	awsRegion := regionFlag
	taskDefinitionName := args[0]

	sess := session.New(&aws.Config{Region: awsRegion})
	if *profileFlag != "" {
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
			SharedConfigState:       session.SharedConfigEnable,
			Profile:                 *profileFlag,
			Config:                  aws.Config{Region: awsRegion},
		}))
	}

	svc := ecs.New(sess)
	resp, err := svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(taskDefinitionName),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(exitCodeAWSError)
	}

	task := resp.TaskDefinition
	image := task.ContainerDefinitions[0].Image

	ecrClient := ecr.New(sess)
	input := &ecr.GetAuthorizationTokenInput{}
	result, err := ecrClient.GetAuthorizationToken(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(exitCodeDockerError)
	}

	authData := result.AuthorizationData[0]
	token := authData.AuthorizationToken

	data, err := base64.StdEncoding.DecodeString(*token)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(exitCodeDockerError)
	}

	userpass := strings.Split(string(data), ":")
	if len(userpass) != 2 {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(exitCodeDockerError)
	}

	auth := docker.AuthConfiguration{
		Username:      userpass[0],
		Password:      userpass[1],
		ServerAddress: *authData.ProxyEndpoint,
	}

	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)

	pullOptions := docker.PullImageOptions{
		Repository: *image,
	}

	fmt.Printf("==> Pulling %s \n", *image)
	if *verboseFlag {
		pullOptions.OutputStream = os.Stdout
	}

	err = client.PullImage(pullOptions, auth)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(exitCodeDockerError)
	}

	dockerArgs := []string{"run", "-it", "--rm"}

	// envs
	for _, e := range task.ContainerDefinitions[0].Environment {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", *e.Name, *e.Value))
	}
	dockerArgs = append(dockerArgs, *image)

	// start the container
	cmd := exec.Command("docker", append(dockerArgs, args[1:]...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Start()
	cmd.Wait()
}
