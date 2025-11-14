# Vulntor Documentation

This directory contains the official documentation for Vulntor, an advanced penetration testing and security assessment framework. The documentation is built using [Docusaurus](https://docusaurus.io/), a modern static website generator.

> **Note:** This documentation is automatically synced from the main repository at [https://github.com/vulntor-ai/vulntor/tree/main/docs](https://github.com/vulntor-ai/vulntor/tree/main/docs). Any changes should be made in the original repository.

## What's Inside

The documentation provides comprehensive guides, API references, and tutorials for:

- Getting started with Vulntor
- Configuration and setup
- Plugin development
- Security scanning techniques
- Fingerprinting and detection methodologies
- Best practices and advanced usage

## Installation

```bash
yarn
```

## Local Development

```bash
yarn start
```

This command starts a local development server and opens up a browser window. Most changes are reflected live without having to restart the server.

## Build

```bash
yarn build
```

This command generates static content into the `build` directory and can be served using any static contents hosting service.

## Deployment

Using SSH:

```bash
USE_SSH=true yarn deploy
```

Not using SSH:

```bash
GIT_USER=<Your GitHub username> yarn deploy
```

If you are using GitHub pages for hosting, this command is a convenient way to build the website and push to the `gh-pages` branch.
