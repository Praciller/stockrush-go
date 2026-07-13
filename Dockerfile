FROM golang:1.26.5-alpine AS build
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-X stockrush-go/internal/httpserver.Version=${VERSION} -X stockrush-go/internal/httpserver.Commit=${COMMIT} -X stockrush-go/internal/httpserver.BuildTime=${BUILD_TIME}" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 go build -o /out/loadgen ./cmd/loadgen && \
    CGO_ENABLED=0 go build -o /out/seed ./cmd/seed && \
    CGO_ENABLED=0 go build -o /out/invariant-check ./cmd/invariant-check && \
    CGO_ENABLED=0 go build -o /out/healthcheck ./cmd/healthcheck

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/ /usr/local/bin/
COPY db/migrations ./db/migrations
USER nonroot:nonroot
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=30s --retries=3 CMD ["/usr/local/bin/healthcheck"]
CMD ["/usr/local/bin/api"]
