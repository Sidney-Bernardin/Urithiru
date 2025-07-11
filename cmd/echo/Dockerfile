FROM golang:alpine AS build

WORKDIR /app

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -o echo main.go

# ==========

FROM scratch

COPY --from=build /app/echo .

ENTRYPOINT ["./echo"]
