# syntax=docker/dockerfile:1

# ---- build ----
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# ---- runtime (distroless) ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
COPY --from=build /out/worker /worker
EXPOSE 8080
USER nonroot:nonroot
# CMD (not ENTRYPOINT) so the runtime can select the binary per process:
# compose overrides it (entrypoint /worker) and Fly [processes] sets app=/api,
# worker=/worker on the same image.
CMD ["/api"]
