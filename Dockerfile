FROM google/golang:1.4

MAINTAINER Piotr Zduniak <piotr@zduniak.net>

RUN go get github.com/tools/godep

RUN mkdir -p /gopath/src/github.com/lavab/webhook
ADD . /gopath/src/github.com/lavab/webhook
RUN cd /gopath/src/github.com/lavab/webhook && godep go install

ENTRYPOINT ["/gopath/bin/webhook"]
