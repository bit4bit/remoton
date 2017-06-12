#Dockerfile for remoton-server
#Default listen at https:9934 y http:9933
FROM golang:1.8

WORKDIR /go/src/app
COPY . .
RUN go-wrapper download 
RUN go-wrapper install

EXPOSE 9934
EXPOSE 9933

CMD ["go-wrapper", "run"]
