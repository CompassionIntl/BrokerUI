FROM alpine:latest
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY broker-service broker-service
EXPOSE 80
ENTRYPOINT ["./broker-service"]