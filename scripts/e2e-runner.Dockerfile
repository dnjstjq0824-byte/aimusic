# E2E runner: host lacks psql/redis-cli/ffmpeg, so the suite runs here.
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash curl ca-certificates python3 postgresql-client redis-tools ffmpeg \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /work
ENTRYPOINT ["bash"]
