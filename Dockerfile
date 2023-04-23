# syntax=docker/dockerfile:1

##
## Build the application from source
##

FROM golang:1.20 AS build-stage

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY caching/ ./caching
COPY cmd/ ./cmd
COPY consts/ ./consts
COPY fsutils/ ./fsutils
COPY handlers/ ./handlers
COPY internal/ ./internal
COPY util/ ./util

WORKDIR cmd/

RUN CGO_ENABLED=0 GOOS=linux go build -o /energy ./

##
## Run the tests in the container
##


#FROM build-stage AS run-test-stage
#RUN go test -v ./...

##
## Deploy the application binary into a lean image
##

FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

COPY --from=build-stage /energy /energy

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/energy"]
