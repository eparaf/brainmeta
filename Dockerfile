# ---- build stage ----------------------------------------------------------
FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Production build with Postgres + Redis enabled. Static, stripped binary.
RUN CGO_ENABLED=0 GOOS=linux go build -tags "pgx redis" -ldflags="-s -w" -o /out/brain ./cmd/brain

# ---- runtime stage --------------------------------------------------------
FROM alpine:3.20
RUN adduser -D -u 10001 brain && mkdir -p /data && chown brain /data
USER brain
WORKDIR /app
COPY --from=build /out/brain /app/brain

# Learned-state snapshot lives on a mounted volume so the moat survives restarts.
ENV BRAIN_SNAPSHOT=/data/brain-snapshot.json
VOLUME ["/data"]
EXPOSE 8080

# Lightweight liveness check against the health endpoint.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/app/brain"]
CMD ["serve"]
