package core

import (
	"sync"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestReserveAIRateLimit(t *testing.T) {
	service := NewAIAssistantService(nil)
	conf := config.AIAssistantConf{DailyLimit: 3, UserHourlyLimit: 2}

	assert.True(t, service.reserveRateLimit(1, conf))
	assert.True(t, service.reserveRateLimit(1, conf))
	assert.False(t, service.reserveRateLimit(1, conf), "per-user hourly limit should apply")
	assert.True(t, service.reserveRateLimit(2, conf))
	assert.False(t, service.reserveRateLimit(3, conf), "global daily limit should apply")
}

func TestReserveAIRateLimitConcurrent(t *testing.T) {
	service := NewAIAssistantService(nil)
	conf := config.AIAssistantConf{DailyLimit: 40, UserHourlyLimit: 5}
	var wg sync.WaitGroup
	results := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- service.reserveRateLimit(1, conf)
		}()
	}
	wg.Wait()
	close(results)
	allowed := 0
	for result := range results {
		if result {
			allowed++
		}
	}
	assert.Equal(t, 5, allowed)
}
