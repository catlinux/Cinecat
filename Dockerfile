FROM golang:1.26-bookworm AS build
ARG COMMIT=""
ARG BUILT_AT=""
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=$(cat VERSION) -X main.commit=${COMMIT} -X main.builtAt=${BUILT_AT}" -o /out/cinecatd ./cmd/cinecatd

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg ca-certificates intel-media-va-driver \
	&& rm -rf /var/lib/apt/lists/*
COPY --from=build /out/cinecatd /cinecatd
VOLUME ["/data", "/media"]
ENV CINECAT_DB_PATH=/data/cinecat.db
ENV LIBVA_DRIVER_NAME=iHD
ENV CINECAT_LISTEN_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/cinecatd"]
