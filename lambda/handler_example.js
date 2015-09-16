var child_process = require('child_process');
var http = require('http')

exports.handler = function(event, context) {
    var proc = child_process.spawn('./userapp', [], {
        stdio: 'inherit'
    });

    proc.on('data', function(data) {
        console.log(data.toString())
    });

    proc.on('error', function(code) {
        console.log("proc, err: ", code)
    })

    proc.on('close', function(code) {
        console.log("Process exited with non-zero status code: " + code)
    });

    var method = event.Method
    var headers = event.Headers
    var body = event.Body
    var path = event.Path
    var options = {
        host: 'localhost',
        port: "8081",
        path: path,
        headers: headers,
        method: method
    };

    function tryUntilSuccess(options, callback) {
        var req = http.request(options, function(res) {
            var chunks = [];
            res.on("data", function(chunk) {
                chunks.push(chunk);
            });
            res.on("end", function() {
                var buffer = Buffer.concat(chunks);
                response = {
                    "Body": buffer.toString('base64'),
                    "StatusCode": res.statusCode,
                    "Headers": res.headers
                }
                context.succeed(response)
            });
        });
        req.write(body);
        req.end();

        req.on('error', function(e) {
            console.log("Req, err: ", e)
            setTimeout(function() {
                tryUntilSuccess(options, callback);
            }, 1000)
        });
    }

    tryUntilSuccess(options, function(e) {
        console.log(e)
    })
}

// exports.handler({
//     "Body": "",
//     "Headers": {
//         "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
//         "Accept-Encoding": "gzip, deflate, sdch",
//         "Accept-Language": "en-US,en;q=0.8,zh-TW;q=0.6",
//         "Connection": "keep-alive",
//         "Upgrade-Insecure-Requests": "1",
//         "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/45.0.2454.85 Safari/537.36"
//     },
//     "Method": "GET",
//     "Path": "/",
//     "Host": ""
// }, {
//     done: function(str) {
//         console.log(str)
//     }
// })
