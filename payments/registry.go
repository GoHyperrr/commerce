package payments

import (
	"sync"
)

var (
	providersMu sync.RWMutex
	providers   = make(map[string]PaymentProvider)
)

// RegisterProvider registers a payment provider.
func RegisterProvider(provider PaymentProvider) {
	providersMu.Lock()
	defer providersMu.Unlock()
	providers[provider.ID()] = provider
}

// GetProvider retrieves a payment provider by its identifier.
func GetProvider(id string) (PaymentProvider, bool) {
	providersMu.RLock()
	defer providersMu.RUnlock()
	p, ok := providers[id]
	return p, ok
}

// ListProviders returns a list of registered payment provider identifiers.
func ListProviders() []string {
	providersMu.RLock()
	defer providersMu.RUnlock()
	list := make([]string, 0, len(providers))
	for id := range providers {
		list = append(list, id)
	}
	return list
}
