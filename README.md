This is a utility to develop AWS lambda using golang. It currently only supports events coming from an ALB.

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

Then run 
```
local-lambda --binary main
```
