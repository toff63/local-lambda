package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/urfave/cli"
)

var binary string

func main() {
	app := cli.NewApp()
	app.Name = "local-lambda"
	app.Usage = "Simulates request coming from AWS ALB into your lambda"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "binary",
			Value: "main",
			Usage: "lambda binary",
		},
	}
	app.Action = func(c *cli.Context) error {
		binary = c.String("binary")
		http.HandleFunc("/", LambdaServer)
		fmt.Printf("Starting server on port 8080:  http://localhost:8080 and delegating to %s\n", binary)
		http.ListenAndServe(":8080", nil)
		return nil
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)

	}
}

// LambdaServer is the entrypoint
func LambdaServer(w http.ResponseWriter, r *http.Request) {
	event := buildEvent(r)
	cmd := exec.Command("docker", "run", "--rm", "-v", fmt.Sprintf("%s:/var/task", os.Getenv("PWD")), "lambci/lambda:go1.x", binary, event)

	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	// Wait for the process to finish or kill it after a timeout (whichever happens first):
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(5 * time.Second):
		if err := cmd.Process.Kill(); err != nil {
			log.Fatal("failed to kill process: ", err)
		}
		fmt.Printf("%s\n", b.Bytes())
		log.Println("process killed as timeout reached")
	case err := <-done:
		if err != nil {
			log.Fatalf("process finished with error = %v", err)
		}
		fmt.Printf("%s\n", b.Bytes())
		log.Print("process finished successfully")
	}
}

func buildEvent(r *http.Request) string {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(fmt.Sprintf("Failed to read http request body: %v\n", err))
	}

	event := events.ALBTargetGroupRequest{
		RequestContext: events.ALBTargetGroupRequestContext{
			ELB: events.ELBContext{
				TargetGroupArn: "arn:aws:elasticloadbalancing:region:123456789012:targetgroup/my-target-group/6d0ecf831eec9f09",
			},
		},
		HTTPMethod:      r.Method,
		Path:            r.URL.Path,
		Headers:         eventHeader(r.Header),
		IsBase64Encoded: false,
		Body:            string(body),
	}

	b, err := json.Marshal(event)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate the event: %v\n", err))
	}
	return string(b)
}

func eventHeader(h http.Header) map[string]string {
	r := make(map[string]string)
	for k, v := range h {
		r[k] = v[0]
	}
	return r
}
