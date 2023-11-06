FROM golang:1.21.3

WORKDIR /grpc_project_go

COPY . .

RUN go mod download

RUN go build -o main .

CMD ["./main"]
