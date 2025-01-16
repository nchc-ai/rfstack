FROM golang:1.23-alpine


RUN mkdir /rfstack
WORKDIR /rfstack

# COPY go.mod and go.sum files to the workspace
COPY go.mod .
COPY go.sum .

COPY . .
RUN if [ ! -d "/rfstack/vendor" ]; then  go mod vendor; fi

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo -o bin/app .

FROM alpine:3.21
RUN apk add --no-cache curl

COPY --from=0 /rfstack/bin/app .
COPY --from=0 /rfstack/conf/server-config.json /conf/server-config.json
CMD ["/app", "--logtostderr=true"]
