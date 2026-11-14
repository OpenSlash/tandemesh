FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/tandemesh-plane ./cmd/tandemesh-plane
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/tandemesh ./cmd/tandemesh

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /data
COPY --from=build /out/tandemesh-plane /usr/local/bin/tandemesh-plane
COPY --from=build /out/tandemesh /usr/local/bin/tandemesh
ENV TANDEMESH_LISTEN=:8080 TANDEMESH_DB_PATH=/data/tandemesh.db
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["tandemesh-plane"]