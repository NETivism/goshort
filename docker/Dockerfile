FROM golang:1.20-bookworm as build
WORKDIR /usr/src/app
COPY backend .
RUN go mod download && go mod verify
RUN go build -v -o /usr/src/app/goshort 

FROM debian:bookworm
RUN \
  apt-get update && \
  apt-get install --yes libc-bin libc6
RUN \
  mkdir -p /goshort
WORKDIR /goshort
COPY --from=build /usr/src/app/goshort /usr/local/bin
CMD [ "goshort" ]
