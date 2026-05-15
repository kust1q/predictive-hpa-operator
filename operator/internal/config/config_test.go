package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testInitConfig(t *testing.T) {
	t.Setenv("METRICS_ADDR", ":9090")
	t.Setenv("PROBE_ADDR", ":9091")
	t.Setenv("ENABLE_LEADER_ELECTION", "true")

	err := InitConfig()
	assert.NoError(t, err)

	cfg := Get()
	assert.Equal(t, ":9090", cfg.MetricsAddr)
	assert.Equal(t, ":9091", cfg.ProbeAddr)
	assert.True(t, cfg.EnableLeaderElection)
}

func testInitConfigDefault(t *testing.T) {
	err := InitConfig()
	assert.NoError(t, err)

	cfg := Get()
	assert.Equal(t, ":8080", cfg.MetricsAddr)
	assert.Equal(t, ":8081", cfg.ProbeAddr)
	assert.False(t, cfg.EnableLeaderElection)
}

func TestConfig(t *testing.T) {
	t.Run("CustomValues", testInitConfig)
	t.Run("DefaultValues", testInitConfigDefault)
}
