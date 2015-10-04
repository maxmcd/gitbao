echo "Building router"
GOOS=linux GOARCH=amd64 go build -o router/router router/main.go 
echo "Building gitbao"
GOOS=linux GOARCH=amd64 go build
echo "Uploading files"
scp -i ~/gb.pem gitbao ubuntu@52.21.35.138:/home/ubuntu/new/
scp -i ~/gb.pem router/router ubuntu@52.21.35.138:/home/ubuntu/new/
scp -i ~/gb.pem nginx/gitbao.conf ubuntu@52.21.35.138:/home/ubuntu/gitbao.conf
scp -i ~/gb.pem lambda/handler_example.js ubuntu@52.21.35.138:/home/ubuntu/lambda/handler_example.js
scp -i ~/gb.pem prod-config.json ubuntu@52.21.35.138:/home/ubuntu/config.json
scp -i ~/gb.pem -r public ubuntu@52.21.35.138:/home/ubuntu/
scp -i ~/gb.pem -r templates ubuntu@52.21.35.138:/home/ubuntu/
# ssh -i ~/gb.pem ubuntu@52.21.35.138 './restart.sh'
ssh -i ~/gb.pem ubuntu@52.21.35.138 
