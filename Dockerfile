# Copyright (c) 2024-2028 Fluxor Framework
# All rights reserved.
#
# This source code is proprietary and confidential.
# Unauthorized copying, modification, distribution, or use of this software,
# via any medium is strictly prohibited without the express written permission
# of Fluxor Framework.
#
# This code is provided as an example for demonstration purposes only.
# Redistribution or sharing of this source code is not permitted.
#
# License: Proprietary - All Rights Reserved
# For licensing inquiries, please contact: caokhang91@gmail.com
FROM golang:1.24 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the demo server (cmd/main.go)
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/fluxor-demo ./cmd

FROM gcr.io/distroless/static:nonroot

# OCI labels for GitHub Container Registry
# Update org.opencontainers.image.source with your actual repository URL
LABEL org.opencontainers.image.source="https://github.com/fluxorio/fluxor-homework"
LABEL org.opencontainers.image.description="Fluxor Framework demo server - event-driven runtime framework"
LABEL org.opencontainers.image.licenses="Proprietary"

COPY --from=builder /out/fluxor-demo /app/fluxor-demo

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/fluxor-demo"]
