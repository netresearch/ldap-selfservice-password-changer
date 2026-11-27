#!/bin/sh
# Creates a self-signed CA and an LDAPS server certificate for the development
# OpenLDAP container in the directory given as $1, unless they already exist.
# osixia/openldap v2 reads tls.crt, tls.key and ca.crt from that directory.

set -eu

DIR="${1:-/certs}"

if [ -s "$DIR/tls.crt" ] && [ -s "$DIR/tls.key" ] && [ -s "$DIR/ca.crt" ]; then
    echo "LDAPS certificates already exist in $DIR."
    exit 0
fi

apk add --no-cache openssl >/dev/null

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

openssl req -x509 -newkey rsa:3072 -nodes -days 3650 \
    -subj "/CN=GopherPass development CA" \
    -keyout "$TMP/ca.key" -out "$TMP/ca.crt"

openssl req -newkey rsa:3072 -nodes \
    -subj "/CN=openldap" \
    -keyout "$TMP/tls.key" -out "$TMP/tls.csr"

printf 'subjectAltName=DNS:openldap,DNS:localhost,IP:127.0.0.1\nextendedKeyUsage=serverAuth\n' >"$TMP/ext.cnf"
openssl x509 -req -in "$TMP/tls.csr" -days 3650 \
    -CA "$TMP/ca.crt" -CAkey "$TMP/ca.key" -CAcreateserial \
    -extfile "$TMP/ext.cnf" -out "$TMP/tls.crt"

cp "$TMP/ca.crt" "$TMP/tls.crt" "$TMP/tls.key" "$DIR/"
# slapd runs as the ldap user (uid 911) in osixia/openldap v2.
chown 911:911 "$DIR/tls.key" "$DIR/tls.crt" "$DIR/ca.crt"
chmod 600 "$DIR/tls.key"
chmod 644 "$DIR/tls.crt" "$DIR/ca.crt"

echo "Created LDAPS certificates in $DIR."
