FROM golang:1.14

WORKDIR /go/src/app
COPY . .

RUN go test -c ./ -o crawler_lib_test
RUN cd cmd && go build -o crawler 

CMD ["sh", "-c", "./crawl.sh"]