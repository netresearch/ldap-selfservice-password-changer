# LDAP Selfservice Password Changer

## Developing

Prerequisites:

- Go 1.20+
- Node.js v16+
- PNPM 8+ (`npm i -g pnpm`)

```bash
# Install dependencies
pnpm i

# Edit the package.json's `go:start` task to include the arguments,
# you want to give to the application.
# Required are:
#   --ldap-server
#   --readonly-user
#   --readonly-password
#   --base-dn

# Running normally
pnpm start

# Running in dev mode
#   This will restart the application every time, you make
#   a change.
pnpm dev
```

## License

LDAP Selfservice Password Changer is licensed under the MIT license, for more information please refer to the [included LICENSE file](LICENSE).

## Contributing

Feel free to contribute by creating a Pull Request!

This project uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commit messages and the default `gofmt` and `prettier` formatting rules.
