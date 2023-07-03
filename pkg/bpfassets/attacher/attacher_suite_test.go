//go:build bcc
// +build bcc

package attacher

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAttacher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Attacher Suite")
}

func checkDataCollected(processesData []ProcessBPFMetrics, cpuFreqData map[int32]uint64) {
	// len > 0
	Expect(len(processesData)).To(BeNumerically(">", 0))
	Expect(len(cpuFreqData)).To(BeNumerically(">", 0))

	// freq must have a value
	Expect(cpuFreqData[0]).To(BeNumerically(">", 0))
}

var _ = Describe("BPF attacher test", func() {
	It("should attach bpf module", func() {
		defer Detach()

		_, err := Attach()
		Expect(err).NotTo(HaveOccurred())
		_, err = CollectProcesses()
		Expect(err).NotTo(HaveOccurred())
		_, err = CollectCPUFreq()
		Expect(err).NotTo(HaveOccurred())
		Detach()
	})
})
