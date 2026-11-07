# One image holds everything: the frontend is embedded in the Go binary.
FROM node:24-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
COPY backend/go.mod ../backend/go.mod
RUN npm run build

FROM golang:1.26-alpine AS backend
ARG VERSION=dev
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /src/frontend/build/ internal/web/dist/
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/wokko ./cmd/wokko
RUN mkdir -p /out/data /out/backups

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend /out/wokko /wokko
COPY --from=backend --chown=nonroot:nonroot /out/data /data
COPY --from=backend --chown=nonroot:nonroot /out/backups /backups
ENV DATA_DIR=/data BACKUP_DIR=/backups PORT=8080
EXPOSE 8080
VOLUME ["/data", "/backups"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD ["/wokko", "healthcheck"]
ENTRYPOINT ["/wokko"]
