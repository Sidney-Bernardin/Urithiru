FROM golang:alpine AS build

WORKDIR /app

COPY go.* .
RUN go mod download
COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -o echo ./cmd/echo/.

# ==========

FROM scratch

COPY --from=build /app/echo .

CMD ["./echo"]
