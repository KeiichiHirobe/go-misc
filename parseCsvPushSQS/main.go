package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	sqsSdk "github.com/aws/aws-sdk-go/service/sqs"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type record struct {
	ID        uint64
	UserID    uint64
	TimeStamp time.Time
}

// 50ms/100000record
// 500ms/1000000record
func loadCsv(r io.Reader) (ret []record) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		split := strings.Split(line, ",")
		if len(split) < 3 {
			log.Fatal("invalid")
		}
		id, _ := strconv.ParseUint(split[0], 10, 64)
		userID, _ := strconv.ParseUint(split[1], 10, 64)
		timestamp, _ := time.ParseInLocation("2006-01-02", split[2], time.Local)
		ret = append(ret, record{
			ID:        id,
			UserID:    userID,
			TimeStamp: timestamp,
		})
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err.Error())
	}
	return
}

const NumMessage = 10000

// 70ms/100000record
// 800ms/1000000record
func createDummyFile(name string) io.Reader {
	file, _ := os.CreateTemp("", name)
	log.Printf("FILE: %v\n", file.Name())
	writer := csv.NewWriter(file)
	defer os.Remove(file.Name())
	for i := 0; i < NumMessage; i++ {
		var record []string
		id := strconv.FormatUint(uint64(i), 10)
		userID := strconv.FormatUint(uint64(i), 10)
		timestamp := time.Now().Format("2006-01-02")
		record = append(record, id)
		record = append(record, userID)
		record = append(record, timestamp)
		writer.Write(record)
	}
	writer.Flush()
	file.Seek(0, 0)
	return file
}

func GetQueueURL(client *sqsSdk.SQS, queue *string) (*sqs.GetQueueUrlOutput, error) {
	result, err := client.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: queue,
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// 183s/10000record from my laptop
// 107s/10000record
func SendMsg(client *sqsSdk.SQS, queueURL *string, message string) error {
	_, err := client.SendMessage(&sqs.SendMessageInput{
		DelaySeconds: aws.Int64(10),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"Title": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String("The Whistler"),
			},
			"Author": &sqs.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String("John Grisham"),
			},
			"WeeksOn": &sqs.MessageAttributeValue{
				DataType:    aws.String("Number"),
				StringValue: aws.String("6"),
			},
		},
		MessageBody: aws.String(message),
		QueueUrl:    queueURL,
	})
	if err != nil {
		return err
	}
	return nil
}

// 29s/10000record from my laptop
// 19s/10000record from EC2
func SendBatchMsg(client *sqsSdk.SQS, queueURL *string, messages []string, ids []string) error {
	var entries []*sqs.SendMessageBatchRequestEntry
	for idx, message := range messages {
		entry := sqs.SendMessageBatchRequestEntry{
			DelaySeconds: aws.Int64(10),
			// Id is required for batch request
			Id:          aws.String(ids[idx]),
			MessageBody: aws.String(message),
			MessageAttributes: map[string]*sqs.MessageAttributeValue{
				"Title": &sqs.MessageAttributeValue{
					DataType:    aws.String("String"),
					StringValue: aws.String("The Whistler"),
				},
				"Author": &sqs.MessageAttributeValue{
					DataType:    aws.String("String"),
					StringValue: aws.String("John Grisham"),
				},
				"WeeksOn": &sqs.MessageAttributeValue{
					DataType:    aws.String("Number"),
					StringValue: aws.String("6"),
				},
			},
		}
		entries = append(entries, &entry)
	}
	_, err := client.SendMessageBatch(&sqs.SendMessageBatchInput{
		Entries:  entries,
		QueueUrl: queueURL,
	})
	return err
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	r := createDummyFile("hoge")
	records := loadCsv(r)
	log.Println("loadCsv end")

	// snippet-start:[sqs.go.send_message.args]
	queue := flag.String("q", "", "The name of the queue")
	useBatchRequest := flag.Bool("b", false, "Use batch request")
	flag.Parse()

	if *queue == "" {
		fmt.Println("You must supply the name of a queue (-q QUEUE)")
		return
	}
	endpoint := "https://sqs.ap-northeast-1.amazonaws.com"
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Endpoint: &endpoint,
		},
	}))
	client := sqs.New(sess)
	// Get URL of queue
	urlOutput, err := GetQueueURL(client, queue)
	if err != nil {
		fmt.Println("Got an error getting the queue URL:")
		fmt.Println(err)
		return
	}
	queueURL := urlOutput.QueueUrl
	receiveStart := time.Now()

	if *useBatchRequest {
		fmt.Println("Using a BatchRequest")
		// max size is 10
		var batchMessages = make([]string, 0, 10)
		var batchIds = make([]string, 0, 10)
		for idx, record := range records {
			body, err := json.Marshal(record)
			if err != nil {
				log.Printf("Got an error marshal message: %\n", err)
				continue
			}
			if idx%10 == 9 {
				batchMessages = append(batchMessages, string(body))
				batchIds = append(batchIds, strconv.FormatUint(record.ID, 10))
				if err := SendBatchMsg(client, queueURL, batchMessages, batchIds); err != nil {
					log.Printf("Got an error sending the message: %\n", err)
				}
				// reset and reuse memory
				batchMessages = batchMessages[:0]
				batchIds = batchIds[:0]
			} else {
				batchMessages = append(batchMessages, string(body))
				batchIds = append(batchIds, strconv.FormatUint(record.ID, 10))
			}
		}
		if len(batchMessages) != 0 {
			if err := SendBatchMsg(client, queueURL, batchMessages, batchIds); err != nil {
				log.Printf("Got an error sending the message: %\n", err)
			}
		}
	} else {
		for _, record := range records {
			body, err := json.Marshal(record)
			if err != nil {
				log.Printf("Got an error marshal message: %\n", err)
				continue
			}
			if err := SendMsg(client, queueURL, string(body)); err != nil {
				log.Printf("Got an error sending the message: %\n", err)
			}
		}
	}
	log.Printf("%v ms \n", time.Since(receiveStart).Milliseconds())
	log.Printf("finished: %v records \n", len(records))
}
