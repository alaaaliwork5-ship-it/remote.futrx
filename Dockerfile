FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /remote ./cmd/remote

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates git tmux lxc curl gnupg \
    && rm -rf /var/lib/apt/lists/*

# Download the NodeSource setup script to a file first so a curl failure fails
# the build loudly instead of silently feeding an empty stream to bash.
RUN curl -fsSL https://deb.nodesource.com/setup_22.x -o /tmp/setup-node.sh \
    && bash /tmp/setup-node.sh \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -f /tmp/setup-node.sh \
    && rm -rf /var/lib/apt/lists/*

# npm installs run without a tail pipe so a registry failure fails the build
# loudly; the npm ls at the end verifies every host CLI actually landed.
RUN npm install -g \
    @anthropic-ai/claude-code@2.1.216 \
    @openai/codex@0.145.0 \
    @moonshot-ai/kimi-code@0.28.1 \
    opencode-ai@1.18.18 \
    && npm ls -g --depth=0

# Guard against the Debian-default node 18 fallback: the pinned agents require
# node 22+, and a silent downgrade breaks their auth flows.
RUN node --version | grep -q '^v22' \
    || { echo "node 22 required (NodeSource setup failed or downgraded)"; exit 1; }

RUN mkdir -p /opt/remote.futrx/data /var/lib/remote/projects

COPY --from=builder /remote /usr/local/bin/remote

ENV HOST=0.0.0.0
ENV PORT=7682
ENV DATA_DIR=/opt/remote.futrx/data
ENV INSTALL_DIR=/opt/remote.futrx
ENV BASE_URL=http://localhost:7682

EXPOSE 7682

CMD ["remote"]
