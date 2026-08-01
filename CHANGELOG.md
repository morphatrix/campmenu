# Changelog

All notable changes to this project are documented here.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
versioning follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-08-01

First tagged release. CampMenu was already in daily use before this point;
this tag marks the start of versioned releases going forward.

### Added
- `campmenu-dev` staging namespace and `dev` branch, so changes can be
  validated in a real deployed environment before reaching `main`/prod.
- Ingredient autocomplete (fuzzy suggestions against the shared ingredient
  referential) on the desktop matrix "new article" input.
- New lists created from within an event default to "event-specific" scope.

### Fixed
- Ad-hoc/mobile shopping items were being saved as voted lists by default
  (GORM zero-value gotcha).
