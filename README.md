# AWS SQS Worker using Goroutine
`sqs-worker-go` is a package that makes consuming SQS messages easier. It supports message deletion after process and backup to a sepcified Firehose stream.

## Installing
### Using Glide
Glide is a dependency management tool (think of it as npm) that downloads the required packages and install them into your local development envirnment for Go to reference.

Simply include the package in your code

```
import (
	"github.com/bufferapp/sqs-worker-go/worker"
)
```
and run `glide create`. You will be promted for questions like version locking. The comamnd will generate a `glide.yaml` file that is populated with the dependencies in your code. Once finished, run `glide install` to install the dependencies from `glide.yaml`

### Using `go get`
Simply run this command to install the package and its underlying packages to your Go environment.
`go get -u github.com/bufferapp/sqs-worker-go/worker`

## Using `sqs-worker-go`
It's a simple create and start, then you are all set.

```
// Creating a new worker service
w, err := worker.NewService("your sqs queue name")
	if err != nil {
		log.Printf("Error creating new Worker Service: %s\n", err)
	}

// Start the worker
// m.process is the function you would like to use to process each individual
// SQS message
w.Start(worker.HandlerFunc(Process))

// Start the worker with backup option
w.Backup("your backup firehose name").Start(worker.HandlerFunc(Process))
```
Please noe `Process` function should have the signature of `func(msg *sqs.Message) error`, while msg is the message for processing.

### Examples
* [elasticserach-indexer-worker](https://github.com/bufferapp/elasticserach-indexer-worker/blob/master/main.go)

### More options
To control number of goroutine (`sqs-worker-go` will assign each mesage to a new goroutine), we could control the number of SQS messages for each batch. The option is available through exported variables.

```
// Assume worker service is created as w
w.MaxNumberOfMessage = 20 // Default is 10 messages
// Wait time for each poll
w.WaitTimeSecond = 30 // Default is 20 seconds
