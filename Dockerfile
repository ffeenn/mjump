FROM golang:1.20.5-alpine3.18
COPY mjump /mjump
COPY config.json /config.json
WORKDIR /
ENTRYPOINT ["./mjump"]