# Changesets

This project uses Changesets for version management and publishing.

## Adding a Changeset

When making changes that should result in a new version:

```bash
pnpm changeset
```

This will prompt you to:
1. Select which packages have changed
2. Choose the type of version bump (major/minor/patch)
3. Write a summary of the changes

## Release Process

1. Changesets accumulate in `.changeset/` directory
2. Run `pnpm version` to consume changesets and update versions/changelogs
3. Run `pnpm release` to build and publish to npm
