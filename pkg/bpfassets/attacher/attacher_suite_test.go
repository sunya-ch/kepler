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
		defer detachBccModule()
		defer detachLibbpfModule()

		_, err := attachBccModule()
		// if build with -tags=bcc, the bpf module will be attached successfully
		Expect(err).NotTo(HaveOccurred())
		_, err = bccCollectProcess()
		Expect(err).NotTo(HaveOccurred())
		_, err = bccCollectFreq()
		Expect(err).NotTo(HaveOccurred())
		detachBccModule()

		_, err = attachLibbpfModule()
		// if build with -tags=libbpf, the bpf module will be attached successfully
		Expect(err).NotTo(HaveOccurred())
		processesData, err := libbpfCollectProcess()
		Expect(err).NotTo(HaveOccurred())
		cpuFreqData, err := libbpfCollectFreq()
		Expect(err).NotTo(HaveOccurred())
		checkDataCollected(processesData, cpuFreqData)
		detachLibbpfModule()
	})
})
