package model

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	usageMetrics   = []string{"bytes_read", "bytes_writes", "cache_miss", "cgroupfs_cpu_usage_us", "cgroupfs_memory_usage_bytes", "cgroupfs_system_cpu_usage_us", "cgroupfs_user_cpu_usage_us", "cpu_cycles", "cpu_instr", "cpu_time"}
	systemFeatures = []string{"cpu_architecture"}
	usageValues    = [][]float64{{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 1, 1, 1, 1, 1, 1, 1, 1}}
	nodeUsageValue = usageValues[0]
	systemValues   = []string{"Sandy Bridge"}
	empty          = []float64{}
)

func TestModels(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Model Suite")
}

var _ = BeforeSuite(func() {
	InitMetricIndexes(usageMetrics)
})

var _ = AfterSuite(func() {
})
