FROM golang:1.17-buster AS build
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -tags netgo -o bucketboy ./cmd/...

FROM alpine:latest
COPY --from=build /build/bucketboy /usr/bin/bucketboy
RUN chmod +x /usr/bin/bucketboy