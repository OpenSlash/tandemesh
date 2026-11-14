FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/orca-plane ./cmd/orca-plane

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/orca-plane /orca-plane
EXPOSE 8080
ENTRYPOINT ["/orca-plane"]
