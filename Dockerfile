FROM golang:1.17


WORKDIR /go/src/naver
COPY . .

ENV GOPROXY=https://goproxy.io,direct

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["naver"]
