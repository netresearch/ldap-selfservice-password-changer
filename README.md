# LDAP Selfservice Password Changer

## Developing

Prerequisites:

- Go 1.20+

```bash
# Run once
go run .

# Run and watch for changes
go install github.com/mitranim/gow@latest
gow run .
```

### Updating the styles or HTML

If you have to change any of the HTML, you'll have to regenerate the style sheet.

Prerequisites:

- Node.js v16+
- PNPM

```bash
# Install dependencies
pnpm i

# Build once
pnpm css

# Build and watch for changes
pnpm css:dev
```

## License

LDAP Selfservice Password Changer is licensed under the MIT license, for more information please refer to the [included LICENSE file](LICENSE).

## Contributing

Feel free to contribute by creating a Pull Request!

This project uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commit messages and the default `gofmt` and `prettier` formatting rules.
