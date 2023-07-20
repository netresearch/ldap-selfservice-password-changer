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
COPY --from=frontend-builder /build/internal/web/static/js/*.js /build/internal/web/static/js
RUN CGO_ENABLED=0 go build -o /build/ldap-passwd

FROM alpine:3 AS runner

COPY --from=backend-builder /build/ldap-passwd /usr/local/bin/ldap-passwd

ENTRYPOINT [ "/usr/local/bin/ldap-passwd" ]
