FROM mcr.microsoft.com/oss/go/microsoft/golang:1.23 as builder

WORKDIR /go/src/AppController
ADD . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -a -ldflags '-extldflags "-static"' -o app-controller

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /go/src/AppController/app-controller .
COPY --from=builder /go/src/AppController/config/crd/bases ./crd
ENTRYPOINT ["/app-controller"]