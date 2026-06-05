# commerce — Hyperrr Decoupled Commerce Module

[![Go Reference](https://pkg.go.dev/badge/github.com/GoHyperrr/commerce.svg)](https://pkg.go.dev/github.com/GoHyperrr/commerce)
[![Go Coverage](https://github.com/GoHyperrr/commerce/wiki/coverage.svg)](https://raw.githack.com/wiki/GoHyperrr/commerce/coverage.html)

This repository contains high-performance, decoupled commerce sub-modules for catalog management, shopping carts, checkout workflows, fulfillment, customer accounts, search, and customer support.

---

## 📦 Sub-Modules

* **`product`**: Product catalog management and price/metadata validation.
* **`cart`**: Active customer shopping cart maintenance and validation.
* **`order`**: Orchestrates checkout, payment reservation, and shipment workflow DAGs.
* **`fulfillment`**: Pluggable inventory checkouts and shipment details tracking.
* **`finance`**: Pluggable transaction processing and saga rollback/refund compensations.
* **`marketing`**: Coupon code calculations and customer loyalty point rewards.
* **`notification`**: Dynamic email, SMS, and push notifications routing.
* **`search`**: Low-latency, full-text product search indexing.
* **`customer`**: Customer profiling, segmentation workflows, and AI-driven personas.
* **`support`**: Ticket management and autonomous AI support agent dispatch.

---

## 🧪 Testing

All modules are 100% decoupled from the core `hyperrr` execution engine at compile time. Unit tests are executed against SQLite in-memory databases using `mdk.TestRuntime`.

To run all tests locally:
```bash
go test ./...
```
