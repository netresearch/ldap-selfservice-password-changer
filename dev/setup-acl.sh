#!/bin/bash
# Wait for OpenLDAP to be ready, then apply ACLs that let each user change
# their own userPassword. Mirrors the AD default where users can change
# their own password.
#
# Patterned after netresearch/ldap-manager dev stack.
set -e

LDAP_HOST="${1:-ldap://openldap:389}"
MAX_RETRIES=30

echo "Waiting for OpenLDAP at $LDAP_HOST..."
for i in $(seq 1 $MAX_RETRIES); do
    if ldapsearch -x -H "$LDAP_HOST" -b "" -s base namingContexts >/dev/null 2>&1; then
        echo "OpenLDAP is ready."
        break
    fi
    if [ "$i" = "$MAX_RETRIES" ]; then
        echo "OpenLDAP not ready after $MAX_RETRIES attempts, giving up."
        exit 1
    fi
    sleep 1
done

echo "Applying ACLs..."
ldapmodify -H "$LDAP_HOST" -D "cn=admin,cn=config" -w config <<'EOF'
dn: olcDatabase={1}mdb,cn=config
changetype: modify
replace: olcAccess
olcAccess: {0}to attrs=userPassword by self write by anonymous auth by * none
olcAccess: {1}to * by users read by * none
EOF

echo "ACLs applied successfully."
