FROM golang:1.15-alpine AS build

WORKDIR /go/src/github.com/koelschkellerkind/owtracecollector
COPY . .
WORKDIR /go/src/github.com/koelschkellerkind/owtracecollector/cmd
RUN go build -o ../bin/owtracecollector

FROM alpine:latest
COPY --from=build /go/src/github.com/koelschkellerkind/owtracecollector/bin/owtracecollector /bin/owtracecollector
ENTRYPOINT ["/bin/owtracecollector"]

EXPOSE 55680 8888