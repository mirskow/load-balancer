FROM golang:1.23-alpine

WORKDIR /app
COPY . .

RUN go build -o /build ./cmd \
    && go clean -cache -modcache

EXPOSE 8080

CMD ["/build"]