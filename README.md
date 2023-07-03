# LDAP Selfservice Password Changer

## Developing

Prerequisites:

- Go 1.20+
- Node.js v16+
- PNPM

```bash
# Install dependencies
pnpm i

# Build CSS once
# If you change any of the HTML, you'll have to regenerate the style sheet.
pnpm css
# Build JS once
# If you change any of the TypeScript code, you'll have to recompile the JS files.
pnpm js
# Run once
go run .

# Build CSS and watch for changes
pnpm css:dev

# Build JS and watch for changes
pnpm js:dev

# Run and watch for changes
go install github.com/mitranim/gow@latest
gow run .
```

## License

LDAP Selfservice Password Changer is licensed under the MIT license, for more information please refer to the [included LICENSE file](LICENSE).

## Contributing

Feel free to contribute by creating a Pull Request!

This project uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commit messages and the default `gofmt` and `prettier` formatting rules.
