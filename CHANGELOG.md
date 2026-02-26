# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2026-02-26

### Tipo de Release: patch

- chore: upgrade edugo-shared/auth to v0.51.0
- fix: correct swagger response type for GetScreenVersion endpoint
- fix: address code review — error handling, deterministic responses, swagger
- feat: Sprint 8 — sync bundle, deterministic hashes, screen versioning

---

## [0.3.0] - 2026-02-26

### Tipo de Release: patch

- fix: address code review issues from PR#5 search/filter feature
- Add search and filtering support to list endpoints and update dependencies

---

## [0.2.0] - 2026-02-25

### Tipo de Release: patch

- Remove unused fields from CreateInstanceRequest and docs

---

## [0.1.2] - 2026-02-24

### Tipo de Release: patch

- fix: allow super_admin to switch context without school membership
- fix: use GITHUB_TOKEN instead of GHCR_TOKEN for registry auth

---

## [0.1.1] - 2026-02-24

### Tipo de Release: patch



---

## [0.1.0] - 2026-02-24

### Tipo de Release: patch

- Remove config directory copy from Dockerfile
- Update edugo-shared/repository to use v0.1.0 module
- chore: release v0.1.0
- Use secrets for Postgres config in Azure deploy workflow
- Update dependencies and remove unused hashToken function
- docs: add CHANGELOG and version tracking
- feat: add GitHub Actions CI/CD workflows
- feat: add Docker and Make build configuration
- feat: initial commit - IAM Platform API

---

## [0.1.0] - 2026-02-24

### Tipo de Release: patch

- Use secrets for Postgres config in Azure deploy workflow
- Update dependencies and remove unused hashToken function
- docs: add CHANGELOG and version tracking
- feat: add GitHub Actions CI/CD workflows
- feat: add Docker and Make build configuration
- feat: initial commit - IAM Platform API

---

---

## [Unreleased]

### Added
- Initial project structure
- Authentication and authorization modules
- User, role, and permission management
- Screen configuration and menu management
- PostgreSQL persistence layer
- Swagger documentation
- CORS middleware and error handling
- Docker support with multi-stage builds
- Makefile for development and CI/CD
- GitHub Actions workflows for testing and deployment
