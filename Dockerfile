FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /duet ./cmd/duet && mkdir /data

FROM scratch
COPY --from=build /duet /duet
COPY --from=build --chown=1000:1000 /data /data
USER 1000
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s CMD ["/duet", "-health"]
VOLUME /data
ENTRYPOINT ["/duet"]
