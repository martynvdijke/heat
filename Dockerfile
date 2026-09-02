FROM node:24-alpine AS ts-builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY tsconfig.json ./
COPY build.mjs ./
COPY ts/ ts/
COPY static/css static/css
COPY static/sw.template.js static/
RUN mkdir -p static/js && node build.mjs

FROM golang:1.27.1-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=ts-builder /app/static/vendor ./static/vendor
COPY --from=ts-builder /app/static/js ./static/js
COPY --from=ts-builder /app/static/sw.js ./static/sw.js

RUN CGO_ENABLED=1 GOOS=linux go build -o heat-server .

FROM alpine:latest
RUN apk add --no-cache sqlite-libs ca-certificates

WORKDIR /app

ENV DOCKER=true

COPY --from=builder /app/heat-server .
COPY --from=builder /app/static ./static

RUN mkdir -p /db /app/media && chmod 777 /db /app/media

EXPOSE 6270

CMD ["./heat-server"]