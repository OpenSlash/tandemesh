FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/orca-plane ./cmd/orca-plane
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/orca ./cmd/orca

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /data
COPY --from=build /out/orca-plane /usr/local/bin/orca-plane
COPY --from=build /out/orca /usr/local/bin/orca
ENV ORCA_LISTEN=:8080 ORCA_DB_PATH=/data/orca.db
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["orca-plane"]