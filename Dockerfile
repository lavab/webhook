FROM google/golang

RUN go get github.com/tools/godep

RUN mkdir -p /gopath/src/github.com/lavab/webhook
ADD . /gopath/src/github.com/lavab/webhook
RUN cd /gopath/src/github.com/lavab/webhook && godep go install

CMD []
ENTRYPOINT ["/gopath/bin/webhook"]
