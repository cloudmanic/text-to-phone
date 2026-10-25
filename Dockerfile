# =============================================================================
# File: Dockerfile
# Author: Spicer Matthews <spicer@cloudmanic.com>
# Created: 2026-10-24
# Copyright: 2026 Cloudmanic, LLC. All rights reserved.
# =============================================================================

# Build a static binary, then ship it in an empty image with only the CA
# certificates needed to reach Twilio over HTTPS. Small image, nothing else to patch.
FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /text-to-phone .

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /text-to-phone /text-to-phone
USER 65534:65534
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 CMD ["/text-to-phone", "healthcheck"]
ENTRYPOINT ["/text-to-phone"]
