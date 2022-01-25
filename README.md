# Pulumi Registry Generation Tool

This tool is used to generate both api docs and package metadata for the Pulumi Registry. This tool calls the docs 
generator package in `pulumi/pulumi` which uses the Pulumi schema for a package to generate API (resource) docs.

## Installation

You can install the `resourcedocsgen` tool just like any other Go-based CLI tool:

```
go install github.com/pulumi/registrygen@master
```

To build and install from source:

```
go build .
```

To install from homebrew:

```bash
brew tap pulumi/tap
brew install registrygen
```

You can also download packages from GitHub Releases.

## Usage

Then you can run any of the available commands using `registrygen <command> <flags>`. Run `registrygen --help` to see the available commands.

As of this writing, the tool supports two main purposes:

* Generate the Pulumi Package metadata for use in the registry
* Generate API docs and the package nav tree

### Generating package metadata

Package metadata is used by the [Pulumi Registry](https://github.com/pulumi/registry) to generate the listing shown at https://pulumi.com/registry.
The metadata file contains information sourced from the package's own Pulumi schema. The `metadata` command can be invoked via the
following command:

```bash
$ registrygen metadata --repoSlug pulumi/pulumi-aws --version v4.34.0 --schemaFile=provider/cmd/pulumi-resource-aws/schema.json
```

The available parameters can be found as follows:

```bash
$ registrygen metadata --help
Generate package metadata from Pulumi schema

Usage:
  registrygen metadata <args> [flags]

Flags:
      --category string         The category for the package. Value must match one of the keys in the map: map[cloud:Cloud database:Database infrastructure:Infrastructure monitoring:Monitoring network:Network utility:Utility vcs:Version Control System]
      --component               Whether or not this package is a component and not a provider
  -h, --help                    help for metadata
      --metadataDir string      The location to save the metadata - this will default to the folder structure that the registry expects (themes/default/data/registry/packages)
      --packageDocsDir string   The location to save the package docs - this will default to the folder structure that the registry expects (themes/default/data/registry/packages)
      --publisher string        The publisher's display name to be shown in the package. This will default to Pulumi
      --repoSlug string         The repository slug e.g. pulumi/pulumi-provider
  -s, --schemaFile string       Relative path to the schema.json file from the root of the repository
      --title string            The display name of the package. If ommitted, the name of the package will be used
      --version string          The version of the package
```

### Generating API docs and the package nav tree

Package API docs are used by the Pulumi Registry as part of the package listing. The api docs are source from the Package schema.
The `docs` command can be invoked via the following command:

```bash
registrygen docs --repoSlug pulumi/pulumi-aws --version v4.34.0 --schemaFile=provider/cmd/pulumi-resource-aws/schema.json --docsOutDir output/api-docs --packageTreeJSONOutDir output/navs
```

The available parameters can be found as follows:

```bash
$ registrygen docs --help
Generate API Docs docs from a Pulumi schema file

Usage:
  registrygen docs [flags]

Flags:
      --docsOutDir string              The directory path to where the docs will be written to
  -h, --help                           help for docs
      --packageTreeJSONOutDir string   The directory path to write the package tree JSON file to
      --repoSlug string                The repository slug e.g. pulumi/pulumi-provider
  -s, --schemaFile string              Path to the schema.json file
      --version string                 The version of the package
```

### The API Docs Templates

This tool depends on the `pulumi/pulumi` repo, namely the `pkg/codegen/docs` generator.
The docs generator uses Go-based [templates](https://github.com/pulumi/pulumi/tree/master/pkg/codegen/docs/templates) to 
render the markdown files in-memory which this tool then writes to the filesystem. All changes to the templates must be
made in the `pulumi/pulumi` repo.
