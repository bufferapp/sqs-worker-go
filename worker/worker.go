package worker

import (
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// HandlerFunc is used to define the Handler that is run on for each message
type HandlerFunc func(msg *sqs.Message) error

// HandleMessage is used for the actual execution of each message
func (f HandlerFunc) HandleMessage(msg *sqs.Message) error {
	return f(msg)
}

// Handler interface
type Handler interface {
	HandleMessage(msg *sqs.Message) error
}

// InvalidMessageError for message that can't be processed and should be deleted
type InvalidMessageError struct {
	SQSMessage string
	LogMessage string
}

func (e InvalidMessageError) Error() string {
	return fmt.Sprintf("[Invalid Message: %s] %s", e.SQSMessage, e.LogMessage)
}

// NewInvalidMessageError to create new error for messages that should be deleted
func NewInvalidMessageError(SQSMessage, logMessage string) InvalidMessageError {
	return InvalidMessageError{SQSMessage: SQSMessage, LogMessage: logMessage}
}

// Service works through the job SQS queue
type Service struct {
	AWSSession         *session.Session
	BackupFirehose     *firehose.Firehose
	BackupFirehoseName string
	JobSQS             *sqs.SQS
	JobSQSURL          string
}

// Exported variables
var (
	// MaxNumberOfMessage at one poll
	MaxNumberOfMessage int64 = 10
	// WaitTimeSecond for each poll
	WaitTimeSecond int64 = 20
)

// Backup to set up Firehose backup option
func (s *Service) Backup(n string) *Service {
	s.BackupFirehose = firehose.New(s.AWSSession)
	s.BackupFirehoseName = n

	return s
}

// NewService creates new worker service
func NewService(n string) (*Service, error) {
	// Setting up SQS connection
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	s := sqs.New(sess)

	resultURL, err := s.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(n),
	})
	if err != nil {
		log.Println("Can't get the SQS queue")
		return nil, err
	}

	builder := &Service{
		AWSSession: sess,
		JobSQS:     s,
		JobSQSURL:  aws.StringValue(resultURL.QueueUrl),
	}

	return builder, nil
}

// Start starts the polling and will continue polling till the application is forcibly stopped
func (s *Service) Start(h Handler) {
	for {
		params := &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(s.JobSQSURL), // Required
			MaxNumberOfMessages: aws.Int64(MaxNumberOfMessage),
			MessageAttributeNames: []*string{
				aws.String("All"), // Required
			},
			WaitTimeSeconds: aws.Int64(WaitTimeSecond),
		}

		resp, err := s.JobSQS.ReceiveMessage(params)
		if err != nil {
			log.Println(err)
			continue
		}
		if len(resp.Messages) > 0 {
			run(s, h, resp.Messages)
		}
	}
}

// poll launches goroutine per received message and wait for all message to be processed
func run(s *Service, h Handler, messages []*sqs.Message) {
	numMessages := len(messages)

	var wg sync.WaitGroup
	wg.Add(numMessages)
	for i := range messages {
		go func(m *sqs.Message) {
			// launch goroutine
			defer wg.Done()
			if err := handleMessage(s, m, h); err != nil {
				log.Println(err.Error())
			}
		}(messages[i])
	}

	wg.Wait()
}

func (s *Service) shouldBackup() bool {
	return (s.BackupFirehose != nil && s.BackupFirehoseName != "")
}

func handleMessage(s *Service, m *sqs.Message, h Handler) error {
	err := h.HandleMessage(m)
	if _, ok := err.(InvalidMessageError); ok {
		// Invalid message encountered. Swallow the error and delete the message
		log.Println(err.Error())
	} else if err != nil {
		// Message is valid but there is an error proccesing it. Keeping it in the
		// queue or send to DLQ to try again
		return err
	}

	// Backup to Firehose option is set
	if s.shouldBackup() {
		params := &firehose.PutRecordInput{
			DeliveryStreamName: aws.String(s.BackupFirehoseName), // Required
			Record: &firehose.Record{ // Required
				Data: []byte(*m.Body),
			},
		}
		_, err = s.BackupFirehose.PutRecord(params)

		if err != nil {
			// Swallow the backup error and go on deleting the message
			log.Println(err.Error())
		}
	}

	// Delete the processed (or invalid) message
	params := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(s.JobSQSURL), // Required
		ReceiptHandle: m.ReceiptHandle,         // Required
	}
	_, err = s.JobSQS.DeleteMessage(params)
	if err != nil {
		return err
	}

	return nil
}
