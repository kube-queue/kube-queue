FROM golang:1.15.3-alpine3.12 as build
RUN apk add make
ADD . /go/src/github.com/kube-queue
WORKDIR /go/src/github.com/kube-queue
RUN make

FROM alpine:3.12
COPY --from=build /go/src/github.com/kube-queue/bin/kube-queue /usr/bin/kube-queue
RUN chmod +x /usr/bin/kube-queue
ENTRYPOINT ["/usr/bin/kube-queue"]