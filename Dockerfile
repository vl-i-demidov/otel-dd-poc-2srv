##
## Build
##
FROM golang:1.18-buster AS build

#
WORKDIR /app

COPY . .
RUN go mod download
RUN go build -o /main-app

##
## Deploy
##
FROM ubuntu:22.04

WORKDIR /

COPY --from=build /main-app /app
COPY ./*.env ./