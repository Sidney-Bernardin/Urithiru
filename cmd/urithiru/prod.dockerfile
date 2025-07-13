FROM golang:alpine AS build

ARG BUILD_TAGS=""
ENV BUILD_TAGS=$BUILD_TAGS

WORKDIR /app

COPY go.* .
RUN go mod download
COPY . .

RUN go build -tags="${BUILD_TAGS}" -o urithiru ./cmd/urithiru/.

# ==========

FROM scratch

ARG CONFIG="default.toml"
ENV CONFIG=$CONFIG

COPY --from=build /app/urithiru .
COPY --from=build /app/${CONFIG} /etc/urithiru/config.toml

CMD [ "./urithiru" ]
