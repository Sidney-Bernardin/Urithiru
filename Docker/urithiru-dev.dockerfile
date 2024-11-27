FROM golang:alpine AS build

ARG BUILD_TAGS=""
ENV BUILD_TAGS=$BUILD_TAGS

WORKDIR /app

# Copy and install golang dependencies.
COPY go.* .
RUN go mod download

# Copy everything and build.
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -tags="${BUILD_TAGS}" -o urithiru ./cmd/urithiru/.

# ==========

FROM scratch

ARG CONFIG="default.toml"
ENV CONFIG=$CONFIG

# Copy the binary and config file from the build stage.
COPY --from=build /app/urithiru .
COPY --from=build /app/${CONFIG} /etc/urithiru/config.toml

CMD [ "./urithiru" ]
