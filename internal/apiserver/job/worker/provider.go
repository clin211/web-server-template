package worker

import "github.com/google/wire"

// ProviderSet contains worker dependency providers.
var ProviderSet = wire.NewSet(NewWorker)
