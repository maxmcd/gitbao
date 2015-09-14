package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
)

type Args struct {
	Body    string              `json:"Body"`
	Headers map[string][]string `json:"Headers"`
	Method  string              `json:"Method"`
}

// Depreciated. Using node
func main() {
	if len(os.Args) > 0 {
		var args Args
		argsString := os.Args[1]
		err := json.Unmarshal([]byte(argsString), &args)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(args)
		cmd := exec.Command("./userapp")
		err = cmd.Start()
		if err != nil {
			log.Fatal(err)
		}
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin

		for {
			client := &http.Client{}
			req, err := http.NewRequest(args.Method, args.Headers["Url"][0], nil)
			if err != nil {
				fmt.Println(err)
			}
			// req.Header.Add("If-None-Match", `W/"wyzzy"`)
			resp, err := client.Do(req)
			if err != nil {
				fmt.Println(err)
			} else {
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(string(body))
				break
			}
		}
		cmd.Process.Kill()
	}
}
