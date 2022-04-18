package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"log"
)

// ref: https://yoppi.hatenablog.com/entry/2017/12/16/165943
func main() {
	def := "hello-task-definition"
	launchType := "FARGATE"
	cluster := "hello-cluster"
	container := "app"
	sg := []*string{aws.String("sg-02xxxxx")}
	subnets := []*string{
		aws.String("subnet-0xxxxx"),
		aws.String("subnet-0xxxxx"),
		aws.String("subnet-0xxxxx"),
	}
	assignPublicIP := "DISABLED"
	command := []*string{
		aws.String("./job"),
		aws.String("hello"),
		aws.String("exec"),
	}

	input := &ecs.RunTaskInput{
		TaskDefinition: aws.String(def),
		LaunchType:     aws.String(launchType),
		Cluster:        aws.String(cluster),
		Count:          aws.Int64(1),
		NetworkConfiguration: &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				AssignPublicIp: &assignPublicIP,
				SecurityGroups: sg,
				Subnets:        subnets,
			},
		},
		Overrides: &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				{
					Name:    aws.String(container),
					Command: command,
				},
			},
		},
	}
	client := ecs.New(session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})))
	output, err := client.RunTask(input)
	if err != nil {
		log.Printf("RuntaskErr: %v\n", err)
		return
	}
	if len(output.Failures) != 0 {
		for _, v := range output.Failures {
			log.Printf("Failure: %+v\n", v)
		}
		return
	}
	if len(output.Tasks) > 0 {
		log.Printf("Runtask Suceeded: %+v\n", output.Tasks[0])
		return
	}
	log.Println("Both of Failures and Tasks are empty")
}
