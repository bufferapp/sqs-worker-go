package worker

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	QueueName         = "impossible_to_guess_dev_testing_sqs"
	NonExistQueueName = "impossible_to_exist_dev_testing_sqs"
)

var (
	svc *sqs.SQS
)

func setup() {
	session := session.Must(session.NewSession())

	svc = sqs.New(session, aws.NewConfig().WithRegion("us-east-1"))

	params := &sqs.CreateQueueInput{
		QueueName: aws.String(QueueName), // Required
		Attributes: map[string]*string{
			"DelaySeconds": aws.String("0"), // Required
		},
	}
	_, err := svc.CreateQueue(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	// fmt.Println(resp)
}

func teardown() {
	resultURL, err := svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(QueueName),
	})
	params := &sqs.DeleteQueueInput{
		QueueUrl: aws.String(*resultURL.QueueUrl), // Required
	}
	_, err = svc.DeleteQueue(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	// fmt.Println(resp)
}

func TestMain(m *testing.M) {
	setup()
	// Wait for the SQS queue to be created
	// FIXME: We should find a better way doing this. Maybe through mocking
	time.Sleep(15000 * time.Millisecond)
	retCode := m.Run()
	teardown()
	// Wait a bit after the SQS queue is deleted. We cannot crate a SQS queue for
	// for the same name within 60 secounds after it's deleted
	time.Sleep(15000 * time.Millisecond)
	os.Exit(retCode)
}

func TestNewService(t *testing.T) {
	w, err := NewService(QueueName)

	if w == nil {
		t.Errorf("Expected new service, got error %s", err)
	}

	_, err = NewService(NonExistQueueName)

	if err == nil {
		t.Errorf("Expected error creating service, got error is %s", err)
	}
}
