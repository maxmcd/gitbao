GOOS=linux GOARCH=amd64 go build
scp -i ~/gb.pem gitbao ubuntu@52.21.35.138:/home/ubuntu/new/
scp -i ~/gb.pem lambda/handler_example.js ubuntu@52.21.35.138:/home/ubuntu/lambda/handler_example.js
scp -i ~/gb.pem -r public ubuntu@52.21.35.138:/home/ubuntu/
scp -i ~/gb.pem -r templates ubuntu@52.21.35.138:/home/ubuntu/
# ssh -i ~/gb.pem ubuntu@52.21.35.138 './restart.sh'
rm gitbao
ssh -i ~/gb.pem ubuntu@52.21.35.138 
