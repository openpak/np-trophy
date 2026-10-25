# Multi-stage, cross-compiled: the build stage runs on the runner's native arch and Go
# emits the target arch, so linux/arm64 builds need no QEMU.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -mod=vendor -trimpath -ldflags="-s -w" -o /out/np-trophy .

FROM alpine:3.22
LABEL org.opencontainers.image.source="https://github.com/openpak/np-trophy"
RUN addgroup -S app && adduser -S -G app app
USER app
COPY --from=build /out/np-trophy /usr/local/bin/np-trophy
EXPOSE 8530 8531 8443
ENTRYPOINT ["/usr/local/bin/np-trophy"]
