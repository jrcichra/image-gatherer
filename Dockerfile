FROM golang:1.23.2-bullseye as firststage
WORKDIR /image-gatherer
ADD . .
RUN CGO_ENABLED=0 go build -v -o image-gatherer .
FROM gcr.io/distroless/static-debian11
WORKDIR /image-gatherer
COPY --from=firststage /image-gatherer/image-gatherer .
ENTRYPOINT ["/image-gatherer/image-gatherer"]
