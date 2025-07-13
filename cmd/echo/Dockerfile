FROM golang:alpine AS build

WORKDIR /app

COPY go.* .
RUN go mod download
COPY . .

RUN go build -o echo ./cmd/echo/.

# ==========

FROM scratch

COPY --from=build /app/echo .

ENTRYPOINT ["./echo"]
