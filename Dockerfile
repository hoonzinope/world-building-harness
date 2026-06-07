FROM golang:1.23-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/world-tool ./cmd/world-tool

FROM node:22-bookworm-slim

RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates git \
  && rm -rf /var/lib/apt/lists/* \
  && npm install -g @openai/codex@0.128.0

WORKDIR /app
RUN mkdir -p /app/data && chown -R node:node /app/data
COPY --from=build /out/world-tool /usr/local/bin/world-tool
COPY README.md go.mod go.sum ./
COPY docs ./docs
COPY opencrabs ./opencrabs
COPY schema ./schema
COPY scripts ./scripts
COPY packs ./packs

EXPOSE 8097

USER node
ENV WORLD_HARNESS_ADDR=:8097
ENV WORLD_HARNESS_PACKS_ROOT=/app/packs
ENV WORLD_HARNESS_REPO_ROOT=/app
ENV WORLD_HARNESS_DATA_ROOT=/app/data

CMD ["world-tool", "serve", "--addr", ":8097", "--packs-root", "/app/packs"]
