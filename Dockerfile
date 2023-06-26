FROM --platform=amd64 node:18 AS css-builder
WORKDIR /build
RUN npm i -g pnpm

COPY package.json .
COPY pnpm-lock.yaml .
RUN pnpm i

COPY . .

RUN pnpm css

FROM golang:1.20-alpine AS app-builder
WORKDIR /build
RUN apk add git

COPY ./go.mod .
COPY ./go.sum .
RUN go mod download

COPY . .
COPY --from=css-builder /build/internal/web/static/styles.css /build/internal/web/static/styles.css
RUN CGO_ENABLED=0 go build -o /build/ldap-passwd

FROM alpine:3.17 AS runner

COPY --from=app-builder /build/ldap-passwd /usr/local/bin/ldap-passwd

CMD [ "/usr/local/bin/ldap-passwd" ]
