FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -o /out/adminctl ./cmd/adminctl

FROM alpine:3.20
RUN apk add --no-cache ffmpeg ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /out/api /out/worker /out/migrate /out/adminctl /app/
USER app
EXPOSE 8080
CMD ["./api"]
