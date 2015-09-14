# Gitbao

Rewrite of gitbao. Everything is a aws lambda function, or a static file. When an http request is made it triggers the request to be sent to the lambda function, which in turn catches the function and returns it to the server. 

There will likely be a built in latency here of around ??ms, but it will allow for incredible scaling management and permanence of services.

## MVP

### Build

1. Gist code is consumed by application
2. Code is built into go binary, unless errors are found
3. Binary is bundled up in to node.js application
4. Node.js application is pushed into lambda function

### Request

1. Request is made to application
2. Headers are sent to lambda function
3. Function accepts headers.
4. Function makes request to application for request body
5. Application streams request body to function
6. Request is assembled with body, and headers, delivered to Go binary
7. Response is sent to application
8. Application sends response to client

Set up a lambda function that proxies a http request. Test latency, and then build out from there.

```
#STUFF:
$ GOOS=linux GOARCH=amd64 go build test.go
https://gist.github.com/maxmcd/f430e80ab3246b2ae6d9
```


- http://blog.0x82.com/2014/11/24/aws-lambda-functions-in-go/
- https://gist.github.com/miksago/d1c456d4e235e025791d
- http://stackoverflow.com/questions/28718887/node-js-http-request-how-to-detect-response-body-encoding
