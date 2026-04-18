#!/bin/bash
# Wait for OpenLDAP to be ready, then apply ACLs:
#   - each user can change their own userPassword (like the AD default)
#   - the password-reset service account can write any user's userPassword
#     (mirrors AD's admin-driven reset capability)
#   - authenticated users can read everything else
#
# Config is read from environment so compose.yml stays the single source of
# truth. Patterned after netresearch/ldap-manager's dev stack.
set -euo pipefail

LDAP_HOST="${1:-ldap://openldap:389}"
LDAP_CONFIG_PASSWORD="${LDAP_CONFIG_PASSWORD:-config}"
RESET_USER_DN="${RESET_USER_DN:-uid=password-reset,ou=People,dc=netresearch,dc=local}"
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
# Heredoc is unquoted so $RESET_USER_DN is interpolated. The rule order is
# preserved: {0} userPassword rule, then {1} general read. Using `replace:`
# (vs `add:`) guarantees a deterministic ACL regardless of whatever default
# rules the openldap image ships — matching ldap-manager's pattern.
ldapmodify -H "$LDAP_HOST" -D "cn=admin,cn=config" -w "$LDAP_CONFIG_PASSWORD" <<EOF
dn: olcDatabase={1}mdb,cn=config
changetype: modify
replace: olcAccess
olcAccess: {0}to attrs=userPassword by self write by dn.exact="$RESET_USER_DN" write by anonymous auth by * none
olcAccess: {1}to * by users read by * none
EOF

echo "ACLs applied successfully."
