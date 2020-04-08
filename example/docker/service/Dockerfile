FROM iron/base
MAINTAINER Roshan Ranabhat "roshanranabhat11@gmail.com"

RUN echo "$PWD"
EXPOSE 6767
#WORKDIR /go/src/github.com/gobeam/golang-oauth/example
ADD authService/authService-linux-amd64 /
ADD authService/config/config.ini config/
CMD ["./authService-linux-amd64"]
