# Builder image
FROM golang:1.15-alpine3.13 as builder

ENV GO111MODULE=on

RUN apk add --update git gcc libc-dev libgcc make curl gnupg

WORKDIR /go/src/github.com/dimitrovvlado/pglock/

COPY . .
# Build server
RUN make clean build

# Actual image
FROM alpine:3.13

RUN apk --no-cache add ca-certificates curl bash

WORKDIR /app

COPY --from=builder /go/src/github.com/dimitrovvlado/pglock/pglock ./
EXPOSE 8080

CMD ["./pglock"]
