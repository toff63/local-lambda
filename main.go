package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/urfave/cli"
)

var binary string

type elb struct {
	TargetGroupArn string `json:"targetGroupArn"`
}
type requestContext struct {
	Elb elb `json:"elb"`
}
type event struct {
	RequestContext  requestContext    `json:"requestContext"`
	HTTPMethod      string            `json:"httpMethod"`
	Path            string            `json:"path"`
	Headers         map[string]string `json:"headers"`
	IsBase64Encoded bool              `json:"isBase64Encoded"`
	Body            string            `json:"body"`
}

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
	cmd := exec.Command("docker", "run", "--rm", "-v", fmt.Sprintf("%s:/var/task", os.Getenv("PWD")), "lambci/lambda:go1.x", "main", event)
	o, _ := cmd.CombinedOutput()
	fmt.Printf("%s\n", o)
}

func buildEvent(r *http.Request) string {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(fmt.Sprintf("Failed to read http request body: %v\n", err))
	}

	event := event{
		RequestContext: requestContext{
			Elb: elb{
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
