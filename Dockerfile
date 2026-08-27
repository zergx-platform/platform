# syntax=docker/dockerfile:1
ARG REGISTRY=rucoder-artifact.temp.10.199.64.20.nip.io
FROM ${REGISTRY}/library/node:26-alpine AS web
ARG HTTP_PROXY=http://mihomo.develop.svc.cluster.local:7890
ARG HTTPS_PROXY=http://mihomo.develop.svc.cluster.local:7890
ENV HTTP_PROXY=${HTTP_PROXY} HTTPS_PROXY=${HTTPS_PROXY} \
    NO_PROXY=localhost,127.0.0.1,.svc.cluster.local,.svc,.nip.io
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund
COPY web/src src
COPY web/public public
COPY web/index.html web/vite.config.ts web/tsconfig.json ./
COPY web/schema schema
RUN npm run build

FROM ${REGISTRY}/library/golang:1.26-alpine AS build
ARG HTTP_PROXY=http://mihomo.develop.svc.cluster.local:7890
ARG HTTPS_PROXY=http://mihomo.develop.svc.cluster.local:7890
ENV HTTP_PROXY=${HTTP_PROXY} HTTPS_PROXY=${HTTPS_PROXY} \
    NO_PROXY=localhost,127.0.0.1,.svc.cluster.local,.svc,.nip.io \
    GOINSECURE=forgejo.develop.10.199.64.20.nip.io \
    GOPRIVATE=forgejo.develop.10.199.64.20.nip.io
RUN apk add --no-cache git \
    && git config --global http.sslVerify false \
    && git config --global url."https://root:devpassword@forgejo.develop.10.199.64.20.nip.io/".insteadOf "https://forgejo.develop.10.199.64.20.nip.io/"
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd cmd
COPY internal internal
COPY web/embed.go web/embed.go
COPY --from=web /web/dist web/dist
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /out/gateway-go ./cmd/gateway

FROM ${REGISTRY}/library/alpine:3.24
RUN apk add --no-cache ca-certificates
COPY --from=build /out/gateway-go /usr/local/bin/gateway-go
ENV RUCODER_PORT=8080
EXPOSE 8080
ENTRYPOINT ["gateway-go"]
