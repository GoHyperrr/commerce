# Changelog

All notable changes to this project will be documented in this file.

## [0.2.0] - 2026-06-14

### Added
- Pluggable `payments` sub-module supporting Stripe, Razorpay, and Mock providers.
- Workflow handlers for payment intent creation, verification, and sagas.
- GraphQL queries and mutations for payments module.
- Decoupled `taxonomy`, `store`, `analytics`, and `seo` packages.

## [0.1.0] - 2026-06-05

### Added
- Catalog product variant options, pricing normalizations, and image relations.
- Decoupled commerce/seo and commerce/taxonomy packages.
- Refactored test suites to link to mdk/mdktest.
