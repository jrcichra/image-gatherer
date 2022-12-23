FROM golang:1.19.4-bullseye as firststage
WORKDIR /image-gatherer
ADD . .
RUN CGO_ENABLED=0 go build -v -o image-gatherer .
FROM gcr.io/distroless/static-debian11
WORKDIR /image-gatherer
COPY --from=firststage /image-gatherer/image-gatherer .
CMD ["./image-gatherer"]