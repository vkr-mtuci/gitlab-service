FROM golang:1.23.4 as builder

ENV GOBASE /app
WORKDIR /app
COPY . .
RUN go get ./cmd/...
RUN CGO_ENABLED=0 GOOS=linux go build -o gitlab-service ./cmd/main.go

FROM alpine:latest as app
WORKDIR /root/
COPY --from=builder /app/gitlab-service .
COPY .env .env
ENV SERVER_PORT=8080
EXPOSE 8080

CMD [ "./gitlab-service" ]