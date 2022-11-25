FROM golang:1.19.3-alpine3.16 AS build
WORKDIR /src
# Cache go modules, only need to rerun if things change
COPY go.mod go.sum ./
RUN go mod download && go mod verify
# Build
COPY *.go ./
COPY fsdiff/ ./fsdiff/
RUN go build server.go

# Actual running container
FROM alpine:3.16
COPY --from=build /src/server /usr/bin/server
RUN --mount=type=cache,target=/var/cache/apk \
    apk add tar xz curl nano
ENTRYPOINT [ "server" ]

