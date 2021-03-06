package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/mgo.v2/bson"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"

	"github.com/maxmcd/gitbao/apache"
	"github.com/maxmcd/gitbao/model"
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

var destinations map[string]string

func init() {
	populateDestinations()
}

func populateDestinations() {
	baos := model.GetAllBaos()
	destinations = make(map[string]string)
	for _, bao := range baos {
		destinations[bao.ID.Hex()] = bao.FunctionName
	}
}

func main() {
	loggingHandler := apache.NewApacheLoggingHandler(http.HandlerFunc(handler), os.Stderr)
	server := &http.Server{
		Addr:    ":8001",
		Handler: loggingHandler,
	}
	log.Fatal(server.ListenAndServe())
}

func handlerError(err error, w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
	return
}

func handler(w http.ResponseWriter, r *http.Request) {

	host := r.Host
	host_parts := strings.Split(host, ".")

	var subdomain string
	var functionName string

	if len(host_parts) == 3 {
		subdomain = host_parts[0]
		fmt.Println(subdomain)
		functionName = destinations[subdomain]
	}

	if subdomain == "test" {
		http.Error(w, "all ok", http.StatusOK)
		return
	}

	if functionName == "" && subdomain != "" {
		fmt.Printf("checking for %s in the db\n", subdomain)
		isValidHex := bson.IsObjectIdHex(subdomain)
		if isValidHex == true {
			bao, err := model.GetBaoById(subdomain)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fmt.Println(bao)
			destinations[bao.ID.Hex()] = bao.FunctionName
			functionName = bao.FunctionName
		}
	}

	if functionName == "" {
		http.Error(w, host+" not found", http.StatusNotFound)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		handlerError(err, w)
		return
	}
	var args Args

	args.Body = string(body)

	newHeaders := make(map[string]string)
	for name, content := range r.Header {
		if len(content) > 0 {
			newHeaders[name] = content[0]
		}
	}

	newHeaders["Host"] = r.Host

	args.Headers = newHeaders
	args.Method = r.Method
	args.Path = r.URL.RequestURI()

	reqeust, err := json.Marshal(args)
	if err != nil {
		handlerError(err, w)
		return
	}

	lambdaResponse, err := InvokeLambda(functionName, reqeust)
	if err != nil {
		handlerError(err, w)
		return
	}

	var response Response
	err = json.Unmarshal(lambdaResponse.Payload, &response)
	if err != nil {
		handlerError(err, w)
		return
	}

	for key, value := range response.Headers {
		w.Header().Del(key)
		w.Header().Add(key, value)
	}

	base64logs := fmt.Sprintf("%s", *lambdaResponse.LogResult)
	logdata, err := base64.StdEncoding.DecodeString(base64logs)

	if len(response.Headers) == 0 {
		// empty reponse
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Empty response, maybe there was an error?\n\nAWS Log output:\n\n"))
		w.Write(logdata)
		return
	}

	fmt.Printf("%#v\n", w.Header())
	fmt.Println(response)
	data, err := base64.StdEncoding.DecodeString(response.Body)
	if err != nil {
		handlerError(err, w)
		return
	}
	w.WriteHeader(response.StatusCode)
	w.Write(data)
}

func InvokeLambda(name string, payload []byte) (response *lambda.InvokeOutput, err error) {
	svc := lambda.New(&aws.Config{Region: aws.String("us-east-1")})

	params := &lambda.InvokeInput{
		FunctionName: aws.String(name), // Required
		// ClientContext:  aws.String("String"),
		InvocationType: aws.String("RequestResponse"),
		LogType:        aws.String("Tail"),
		Payload:        payload,
	}
	return svc.Invoke(params)
}
