FROM node:22-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS app
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=web /src/backend/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /viagens .

FROM gcr.io/distroless/static
COPY --from=app /viagens /viagens
ENV DATA_DIR=/data
EXPOSE 8080
ENTRYPOINT ["/viagens"]
CMD ["serve"]
