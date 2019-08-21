package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
	resp, err := execute(event)
	if err != nil {
		log.Fatalf("Failed to execute with error = %v", err)
	}
	for key, value := range resp.Headers {
		w.Header().Set(key, value)
	}
	for key, values := range resp.MultiValueHeaders {
		for _, v := range values {
			w.Header().Set(key, v)
		}
	}
	io.WriteString(w, resp.Body)
	w.WriteHeader(resp.StatusCode)
}

func execute(event string) (events.ALBTargetGroupResponse, error) {
	cmd := exec.Command("docker", "run", "--rm", "-v", fmt.Sprintf("%s:/var/task", os.Getenv("PWD")), "lambci/lambda:go1.x", binary, event)

	var out bytes.Buffer
	cmd.Stdout = &out
	var er bytes.Buffer
	cmd.Stderr = &er

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
		return events.ALBTargetGroupResponse{}, errors.New("Failed to start Process")
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
			return events.ALBTargetGroupResponse{}, errors.New("Failed to kill Process after timeout")
		}
		fmt.Printf("%s\n", er.Bytes())
		fmt.Printf("%s\n", out.Bytes())
		log.Println("process killed as timeout reached")
		return events.ALBTargetGroupResponse{}, errors.New("Process took too long to process")
	case err := <-done:
		if err != nil {
			log.Fatalf("process finished with error = %v", err)
			return events.ALBTargetGroupResponse{}, errors.New("Process took too long to process")
		}
		fmt.Printf("%s\n", er.Bytes())

		o := out.String()
		fmt.Printf("%s\n", o)

		lines := strings.Split(o, "\n")
		j := lines[len(lines)-2]
		var response events.ALBTargetGroupResponse
		err = json.Unmarshal([]byte(j), &response)
		if err != nil {
			return events.ALBTargetGroupResponse{}, fmt.Errorf("Could not parse response %s due to error: %v", j, err)
		}
		return response, nil
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
