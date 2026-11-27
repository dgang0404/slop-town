# Content pack manifests

Bundled packs live under `content/<id>/`. Reserve `mods/<id>/` for locally installed third-party packs. To check the [bundled example](../content/smoke/manifest.json) from the repository root, run:

```sh
go run ./cmd/packcheck ./content/smoke
```

The checker reads one regular file named `manifest.json` from the chosen directory. Use a UTF-8 JSON object up to 64 KiB in size, with each of the following fields present once. Unknown fields and trailing data cause an error.

| Field | Meaning in version 1 |
| --- | --- |
| `manifest_version` | Integer `1` selects this metadata format. |
| `content_api` | Integer `1` identifies the compatible host content contract. |
| `id` | Stable pack ID. Start with a lowercase ASCII letter. Use lowercase letters, digits and single hyphens between segments. |
| `version` | Positive integer pack revision. Increment it when publishing changed content. |
| `dependencies` | Array of other pack IDs. Use `[]` for a standalone pack. Each ID must be unique and different from this pack's ID. |

## Loading and distribution

Treat mod files as untrusted data. The checker opens a directory selected by the caller and reads the manifest under that root. It rejects linked manifests and treats dependency IDs as names. Validation checks declarations only. A future loader must resolve dependencies from locally installed packs, detect missing dependencies and cycles, and verify referenced assets stay within the pack root before activation.

The fixture contains only metadata. Validation runs offline and requires no credentials. Keep service keys in user-controlled settings outside both the pack and version control. For third-party material, include the source, author, license and required notices alongside the files before distributing a pack. The repository's [license notes](../README.md#license) cover project code and documentation.

Community realism packs may replace the foundation’s food-egg supply chain and introduce adult intimacy or character death. This is a design boundary for future mods. [Base-game rules](plans/phase-6.md#base-game-food-and-community-realism-mods)
