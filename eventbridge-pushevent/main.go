package main

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eventbridge"
	"log"
)

//  export AWS_REGION=ap-northeast-1
func main() {
	// All clients require a Session. The Session provides the client with
	// shared configuration such as region, endpoint, and credentials. A
	// Session should be shared where possible to take advantage of
	// configuration and credential caching. See the session package for
	// more information.
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := eventbridge.New(sess)

	type Msg struct {
		Id   int
		Name string
	}

	msg := Msg{
		Id:   111,
		Name: "hirobe",
	}

	detail, _ := json.Marshal(msg)
	log.Println(string(detail))

	entry := &eventbridge.PutEventsRequestEntry{
		Detail:       aws.String(string(detail)),
		EventBusName: aws.String("hirobe-test"),
		Source:       aws.String("dummy"),
		DetailType:   aws.String("dummy"),
	}
	input := &eventbridge.PutEventsInput{
		Entries: []*eventbridge.PutEventsRequestEntry{entry},
	}
	output, err := svc.PutEvents(
		input,
	)
	if err != nil {
		panic(err)
	}
	log.Printf("%+v \n", output)
}
