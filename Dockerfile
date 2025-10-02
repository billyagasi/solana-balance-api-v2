# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.24 AS build
WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build statically (tanpa CGO)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/server ./cmd/server

# ---- Runtime stage (distroless = minimal & non-root) ----
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /app/server /server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
