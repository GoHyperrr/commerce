# commerce — Hyperrr Decoupled Commerce Module

[![Go Reference](https://pkg.go.dev/badge/github.com/GoHyperrr/commerce.svg)](https://pkg.go.dev/github.com/GoHyperrr/commerce)
[![Go Coverage](https://github.com/GoHyperrr/commerce/wiki/coverage.svg)](https://raw.githack.com/wiki/GoHyperrr/commerce/coverage.html)

This repository contains high-performance, decoupled commerce sub-modules for catalog management, shopping carts, checkout workflows, fulfillment, customer accounts, search, and customer support.

---

## 📦 Sub-Modules
 
* **`product`**: Product catalog management and price/metadata validation.
* **`taxonomy`**: Categories, tags, collections, and catalog hierarchies.
* **`cart`**: Active customer shopping cart maintenance and validation.
* **`order`**: Orchestrates checkout, payment reservation, and order status updates.
* **`payments`**: Pluggable transaction processing with Stripe, Razorpay, and Mock gateways, integrated as saga workflows.
* **`store`**: Store configuration, settings, and business profile settings.
* **`customer`**: Customer profiling, shipping/billing address books, and registered user accounts.
* **`seo`**: Search Engine Optimization metadata generators for products, taxonomies, and pages.

---

## 🧪 Testing

All modules are 100% decoupled from the core `hyperrr` execution engine at compile time. Unit tests are executed against SQLite in-memory databases using `mdk.TestRuntime`.

To run all tests locally:
```bash
go test ./...
```
