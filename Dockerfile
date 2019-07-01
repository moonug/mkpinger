FROM golang:alpine as build

RUN apk --no-cache add tzdata git make build-base
RUN go get github.com/golang/dep/cmd/dep
COPY Gopkg.lock Gopkg.toml /go/src/r.strela4g.ru/strela/pinger/
WORKDIR /go/src/r.strela4g.ru/strela/pinger

RUN dep ensure -vendor-only

COPY . /go/src/r.strela4g.ru/strela/pinger/

RUN go build

FROM alpine:latest as production
#FROM golang:alpine as production
RUN apk add --no-cache tzdata ca-certificates
ENV TZ=Europe/Moscow
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
RUN mkdir -p /usr/local/go/lib/time
WORKDIR /
COPY --from=build /go/src/r.strela4g.ru/strela/pinger/pinger .
COPY entrypoint.sh .
ENTRYPOINT ["sh", "/entrypoint.sh"]
