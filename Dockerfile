FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/affluena ./cmd/api

FROM alpine:3.22

WORKDIR /app
COPY --from=build /bin/affluena /app/affluena
COPY migrations /app/migrations

EXPOSE 8080
CMD ["/app/affluena"]
