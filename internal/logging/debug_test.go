package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger_Creation(t *testing.T) {
	// Clear and set test topics
	enabledTopics = map[string]bool{"test": true}

	enabledLog := New("test")
	disabledLog := New("other")

	assert.True(t, enabledLog.Enabled(), "Logger for enabled topic should be enabled")
	assert.False(t, disabledLog.Enabled(), "Logger for disabled topic should be disabled")
}

func TestLogger_AllTopics(t *testing.T) {
	// Test the "all" wildcard
	enabledTopics = map[string]bool{"*": true}

	log1 := New("anything")
	log2 := New("whatever")

	assert.True(t, log1.Enabled(), "All topics should be enabled with wildcard")
	assert.True(t, log2.Enabled(), "All topics should be enabled with wildcard")
}

func TestLogger_NoTopics(t *testing.T) {
	// Test empty topics
	enabledTopics = map[string]bool{}

	log := New("anything")

	assert.False(t, log.Enabled(), "Logger should be disabled when no topics enabled")
}

func BenchmarkLogger_Disabled(b *testing.B) {
	// Benchmark the fast path when logging is disabled
	enabledTopics = map[string]bool{}
	log := New("benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Debug("test message", "key", "value", "number", 42)
	}
}

func BenchmarkLogger_Enabled(b *testing.B) {
	// Benchmark when logging is enabled
	enabledTopics = map[string]bool{"benchmark": true}
	log := New("benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Debug("test message", "key", "value", "number", 42)
	}
}
