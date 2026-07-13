FROM golang:1.26.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 go build -o /out/loadgen ./cmd/loadgen

FROM alpine:3.22
RUN adduser -D -u 10001 stockrush
WORKDIR /app
COPY --from=build /out/ /usr/local/bin/
COPY db/migrations ./db/migrations
USER stockrush
EXPOSE 8080
CMD ["api"]
