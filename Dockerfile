FROM --platform=amd64 node:18 AS frontend-builder
WORKDIR /build
RUN npm i -g pnpm

COPY package.json .
COPY pnpm-lock.yaml .
RUN pnpm i

COPY . .

RUN pnpm build:assets

FROM golang:1.20-alpine AS backend-builder
WORKDIR /build
RUN apk add git

COPY ./go.mod .
COPY ./go.sum .
RUN go mod download

COPY . .
COPY --from=frontend-builder /build/internal/web/static/styles.css /build/internal/web/static/styles.css
COPY --from=frontend-builder /build/internal/web/static/*.js /build/internal/web/static/
RUN CGO_ENABLED=0 go build -o /build/ldap-passwd

FROM alpine:3.17 AS runner

COPY ./docker-entrypoint.sh /entrypoint.sh
COPY --from=backend-builder /build/ldap-passwd /usr/local/bin/ldap-passwd

ENTRYPOINT [ "/entrypoint.sh" ]
