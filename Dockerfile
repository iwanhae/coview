FROM golang:1.25-alpine AS build
WORKDIR /go/src/coview
COPY . .

ENV CGO_ENABLED=0
RUN go build -v -o /go/bin/coview main.go



# Now copy it into our base image.
FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=build /go/bin/coview /coview

COPY config.yaml config.yaml
COPY web web

EXPOSE 8081

VOLUME /data
CMD ["/coview"]