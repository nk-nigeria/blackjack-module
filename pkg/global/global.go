package global

import (
	"sync"
)

// Global variables for bot integration
var (
	globalBotIntegration interface{}
	globalMutex          sync.RWMutex
)

// GetGlobalBotIntegration returns the global bot integration instance
func GetGlobalBotIntegration() interface{} {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalBotIntegration
}

// SetGlobalBotIntegration sets the global bot integration instance
func SetGlobalBotIntegration(botIntegration interface{}) {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	globalBotIntegration = botIntegration
}
