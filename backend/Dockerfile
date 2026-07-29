FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /hubby-api ./cmd/api
RUN CGO_ENABLED=0 go build -o /hubby-migrate ./cmd/migrate

FROM alpine:3.22
RUN adduser -D -u 10001 hubby
USER hubby
COPY --from=builder /hubby-api /hubby-api
COPY --from=builder /hubby-migrate /hubby-migrate
EXPOSE 8080
ENTRYPOINT ["/hubby-api"]
