FROM golang:alpine AS build

WORKDIR /app

# Copy and install golang dependencies.
COPY go.* .
RUN go mod download

# Copy everything and build.
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -o echo ./cmd/echo/.

CMD ["./echo"]
