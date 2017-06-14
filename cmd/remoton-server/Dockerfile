#Dockerfile for remoton-server
#Default listen at https:9934 y http:9933
FROM golang:1.8
LABEL maintainer "Jovany Leandro G.C <bit4bit@riseup.net>"

WORKDIR /go/src/app
COPY main.go . 
RUN go-wrapper download 
RUN go-wrapper install

EXPOSE 9934 9933

ENV REMOTON_TOKEN_AUTH_SERVER "public"

VOLUME ["/remoton-certs"]
CMD ["/usr/local/bin/go-wrapper", "run", "-listen", "0.0.0.0:9934", "-cert", "/remoton-certs/cert.pem", "-key", "/remoton-certs/key.pem"]
