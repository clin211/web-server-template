package job

import "github.com/google/wire"

// ProviderSet contains job package dependency providers.
var ProviderSet = wire.NewSet(NewExecutionRecorder, NewSchedulerTaskStore)
