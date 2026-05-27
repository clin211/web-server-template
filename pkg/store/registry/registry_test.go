package registry

import (
	"sync"
	"testing"
)

func TestRegistryRegisterConcurrent(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{name: "small batch", count: 16},
		{name: "large batch", count: 128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()

			var wg sync.WaitGroup
			wg.Add(tt.count)
			for i := 0; i < tt.count; i++ {
				i := i
				go func() {
					defer wg.Done()
					registry.Register(&struct{ ID int }{ID: i})
				}()
			}
			wg.Wait()

			registry.mu.RLock()
			got := len(registry.models)
			registry.mu.RUnlock()
			if got != tt.count {
				t.Fatalf("registered model count = %d, want %d", got, tt.count)
			}
		})
	}
}
