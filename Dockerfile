FROM golang:1.13 as builder
ADD . /build
WORKDIR /build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o gnmi-server-gen .

FROM alpine
COPY --from=builder /build/gnmi-server-gen /app/
WORKDIR /app
ENTRYPOINT [ "/app/gnmi-server-gen" ]
CMD [ "--help" ]
