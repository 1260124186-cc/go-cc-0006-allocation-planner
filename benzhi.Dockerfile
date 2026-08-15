FROM golang:1.26.2-alpine

WORKDIR /workspace
ENV GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build ./...

CMD ["sh", "-c", "go run ./cmd/planner -action allocate < /workspace/examples/request.json"]
