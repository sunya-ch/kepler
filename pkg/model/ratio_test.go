package model

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sustainable-computing-io/kepler/pkg/power/rapl/source"
)

var _ = Describe("Test Ratio Unit", func() {
	It("GetPodPowerRatio", func() {
		corePower := []float64{10, 10}
		dramPower := []float64{2, 2}
		uncorePower := []float64{1, 1}
		pkgPower := []float64{15, 15}
		totalCorePower, totalDRAMPower, totalUncorePower, totalPkgPower, _ := GetSumDelta(corePower, dramPower, uncorePower, pkgPower, empty)
		Expect(totalCorePower).Should(BeEquivalentTo(20))
		Expect(totalDRAMPower).Should(BeEquivalentTo(4))
		Expect(totalUncorePower).Should(BeEquivalentTo(2))
		Expect(totalPkgPower).Should(BeEquivalentTo(30))
		sumMetricValues := GetSumMetricValues(usageValues)
		nodeComponentPower := source.RAPLPower{
			Core:   totalCorePower,
			Uncore: totalUncorePower,
			DRAM:   totalDRAMPower,
			Pkg:    totalPkgPower,
		}
		otherNodePower := uint64(10)
		componentPowers, otherPodPowers := GetPodPowerRatio(usageValues, otherNodePower, nodeComponentPower, sumMetricValues)
		Expect(len(componentPowers)).Should(Equal(len(usageValues)))
		Expect(len(otherPodPowers)).Should(Equal(len(usageValues)))
		Expect(componentPowers[0].Core).Should(Equal(componentPowers[1].Core))
		Expect(componentPowers[0].Core).Should(BeEquivalentTo(10))
		Expect(componentPowers[0].DRAM).Should(BeEquivalentTo(2))
		Expect(componentPowers[0].Uncore).Should(BeEquivalentTo(1))
		Expect(componentPowers[0].Pkg).Should(BeEquivalentTo(15))
		Expect(otherPodPowers[0]).Should(BeEquivalentTo(5))
	})
})
