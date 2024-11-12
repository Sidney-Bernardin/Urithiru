FROM golang:alpine AS build

ARG BUILD_TAGS=""
ENV BUILD_TAGS=$BUILD_TAGS

WORKDIR /app

COPY go.* .
RUN go mod download
COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -tags="${BUILD_TAGS}" -o urithiru ./cmd/urithiru/.

# ==========

FROM scratch

ARG CONFIG="config.toml"
ENV CONFIG=$CONFIG

COPY --from=build /app/urithiru .
COPY --from=build /app/${CONFIG} /etc/urithiru/config.toml

CMD [ "./urithiru" ]