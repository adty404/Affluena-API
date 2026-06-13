FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/affluena-api ./cmd/api

FROM alpine:3.22

WORKDIR /app
COPY --from=build /bin/affluena-api /app/affluena-api
COPY migrations /app/migrations

EXPOSE 8080
CMD ["/app/affluena-api"]
