This is a tool to simlulate AWS lambda locally. It helps you develop go lambda function that you plan to put behind an Application Load Balancer.

## Prerequisites

You'll need [Docker](https://docs.docker.com/install/#supported-platforms) installed. Your golang code binary will have to be compatible with Linux.

## Usage

```
go get github.com/toff63/local-lambda
```

Then compile the code in go expecting requests coming from an ALB like:

```go
 package main
    
 import (
     "context"
     "fmt"
    
     "github.com/aws/aws-lambda-go/events"
     "github.com/aws/aws-lambda-go/lambda"
 )
    
 func handleRequest(ctx context.Context, request events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
     fmt.Printf("Processing request data for traceId %s.\n", request.Headers["x-amzn-trace-id"])
     fmt.Printf("Body size = %d.\n", len(request.Body))
    
     fmt.Println("Headers:")
     for key, value := range request.Headers {
   fmt.Printf("    %s: %s\n", key, value)
     }
    
     return events.ALBTargetGroupResponse{Body: request.Body, StatusCode: 200, StatusDescription: "200 OK", IsBase64Encoded: false, Headers: map[string]string{}}, nil
 }
    
 func main() {
     lambda.Start(handleRequest)
 } 
```

Then run in a console
```
local-lambda --binary main
```

In another console you can send the http request like:
```
curl http://127.0.0.1:8080/hello
```

The local-lambda will show you logs that you would usually find in AWS Cloudwatch logs:
```
START RequestId: a9f293cc-8ce5-1790-bb74-dee907441c4b Version: $LATEST
Processing request data for traceId .
Body size = 0.
Headers:
    Accept: */*
    User-Agent: curl/7.58.0
END RequestId: a9f293cc-8ce5-1790-bb74-dee907441c4b
REPORT RequestId: a9f293cc-8ce5-1790-bb74-dee907441c4b  Duration: 1.25 ms       Billed Duration: 100 ms Memory Size: 1536 MB    Max Memory Used: 7 MB
{"statusCode":200,"statusDescription":"200 OK","headers":{},"multiValueHeaders":null,"body":"","isBase64Encoded":false}
```

## Further work

This script is just a thin layer on top of [lambci/docker-lambda](https://github.com/lambci/docker-lambda) so it is possible to extend this project to support further languages and type of events.
