#!/bin/bash
export GOOS=linux
export CGO_ENABLED=0

#RabbitMQ for messaging
go get github.com/streadway/amqp;

#install dep
go get -u github.com/golang/dep/cmd/dep

cd authService; go get; go build -o authService-linux-amd64;
#cd authService; dep ensure; go build -o authService-linux-amd64;
cd ..; echo built `pwd`;

cd postService; go get; go build -o postService-linux-amd64;
cd ..; echo built `pwd`;

export GOOS=darwin

#cp healthchecker/healthchecker-linux-amd64 accountservice/
#cp healthchecker/healthchecker-linux-amd64 vipservice/
#cp healthchecker/healthchecker-linux-amd64 imageservice/

docker-compose build
docker-compose up -d

#RabbitMQ add new user and delete guest user
#docker exec rabbitmq.dev rabbitmq-plugins enable rabbitmq_management
docker exec rabbitmq.dev rabbitmqctl add_user bajra bajraHanyo8848
docker exec rabbitmq.dev rabbitmqctl set_user_tags bajra administrator
docker exec rabbitmq.dev rabbitmqctl set_permissions -p / bajra ".*" ".*" ".*"
docker exec rabbitmq.dev rabbitmqctl delete_user guest
