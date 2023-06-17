FROM golang:1.20.5-alpine3.18 as build

WORKDIR /work

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY /src/go.mod /src/go.sum /work
RUN go mod download && go mod verify

COPY /src /work
RUN go build -v -o app

FROM alpine:3.18 as runtime
WORKDIR /work
COPY --from=build /work/app .
CMD ["/work/app"]
