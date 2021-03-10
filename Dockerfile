FROM golang:1.14

WORKDIR /go/src/app
COPY . .

RUN go test -c ./ -o crawler

CMD ["/go/src/app/crawler", "-test.v"]