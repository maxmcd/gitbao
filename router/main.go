package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
)

type Args struct {
	Body    string            `json:"Body"`
	Headers map[string]string `json:"Headers"`
	Method  string            `json:"Method"`
	Path    string            `json:"Path"`
	Host    string            `json:"Host"`
}

type Response struct {
	Body       string
	Headers    map[string]string
	StatusCode int
}

func main() {
	http.ListenAndServe(":8001", http.HandlerFunc(handler))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	var args Args

	args.Body = string(body)

	newHeaders := make(map[string]string)
	for name, content := range r.Header {
		if len(content) > 0 {
			newHeaders[name] = content[0]
		}
	}

	args.Headers = newHeaders
	args.Method = r.Method
	args.Path = r.URL.RequestURI()

	reqeust, err := json.Marshal(args)
	if err != nil {
		log.Fatal(err)
	}
	name := "forBuild410480062"

	err, payload := InvoteLambda(name, reqeust)
	if err != nil {
		log.Fatal(err)
	}

	var response Response
	err = json.Unmarshal(payload, &response)
	if err != nil {
		log.Fatal(err)
	}

	for key, value := range response.Headers {
		w.Header().Del(key)
		w.Header().Add(key, value)
	}
	fmt.Printf("%#v\n", w.Header())
	fmt.Println(response)
	data, err := base64.StdEncoding.DecodeString(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	w.WriteHeader(response.StatusCode)
	w.Write(data)
}

func InvoteLambda(name string, payload []byte) (err error, response []byte) {
	svc := lambda.New(&aws.Config{Region: aws.String("us-east-1")})

	params := &lambda.InvokeInput{
		FunctionName: aws.String(name), // Required
		// ClientContext:  aws.String("String"),
		// InvocationType: aws.String("InvocationType"),
		// LogType:        aws.String("LogType"),
		Payload: payload,
	}
	resp, err := svc.Invoke(params)
	response = resp.Payload

	return
}
