package main

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"log"
	"os"
)

//  export AWS_REGION=ap-northeast-1
//	export TOPIC_ARN=arn:aws:sns:ap-northeast-1:xxxxxxxxx:test-topic-name
func main() {
	// All clients require a Session. The Session provides the client with
	// shared configuration such as region, endpoint, and credentials. A
	// Session should be shared where possible to take advantage of
	// configuration and credential caching. See the session package for
	// more information.
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := sns.New(sess)

	type Msg struct {
		Id   int
		Name string
	}

	msg := Msg{
		Id:   111,
		Name: "hirobe",
	}

	message, _ := json.Marshal(msg)
	log.Println(string(message))

	attributes := map[string]*sns.MessageAttributeValue{
		"MessageName": {
			DataType:    aws.String("String"),
			StringValue: aws.String("sample-message"),
		},
	}

	input := &sns.PublishInput{
		Message:           aws.String(string(message)),
		TopicArn:          aws.String(os.Getenv("TOPIC_ARN")),
		MessageAttributes: attributes,
	}

	output, err := svc.Publish(input)

	if err != nil {
		panic(err)
	}
	log.Printf("%+v \n", output)
}
