package model

import (
	"log"
	"math"

	"github.com/sustainable-computing-io/kepler/pkg/config"
	"github.com/sustainable-computing-io/kepler/pkg/power/rapl/source"
)

var (
	coreMetricIndex    int = -1
	dramMetricIndex    int = -1
	uncoreMetricIndex  int = -1
	generalMetricIndex int = -1
)

func GetSumMetricValues(podMetricValues [][]float64) (sumMetricValues []float64) {
	podNumber := len(podMetricValues)
	for i := 0; i < podNumber; i++ {
		sumUsage := float64(0)
		for _, podMetricValue := range podMetricValues {
			sumUsage += podMetricValue[i]
		}
		sumMetricValues = append(sumMetricValues, sumUsage)
	}
	return
}

func GetSumDelta(corePower, dramPower, uncorePower, pkgPower, gpuPower []float64) (totalCorePower, totalDRAMPower, totalUncorePower, totalPkgPower, totalGPUPower uint64) {
	for i, val := range pkgPower {
		totalCorePower += uint64(corePower[i])
		totalDRAMPower += uint64(dramPower[i])
		totalUncorePower += uint64(uncorePower[i])
		totalPkgPower += uint64(val)
	}
	for _, val := range gpuPower {
		totalGPUPower += uint64(val)
	}
	return
}

func InitMetricIndexes(metricNames []string) {
	for index, metricName := range metricNames {
		if metricName == config.CoreUsageMetric {
			coreMetricIndex = index
			log.Printf("set coreMetricIndex = %d", index)
		}
		if metricName == config.DRAMUsageMetric {
			dramMetricIndex = index
			log.Printf("set dramMetricIndex = %d", index)
		}
		if metricName == config.UncoreUsageMetric {
			uncoreMetricIndex = index
			log.Printf("set uncoreMetricIndex = %d", index)
		}
		if metricName == config.GeneralUsageMetric {
			generalMetricIndex = index
			log.Printf("set generalMetricIndex = %d", index)
		}
	}
}

func getRatio(podMetricValue []float64, metricIndex int, totalPower uint64, podNumber float64, sumMetricValues []float64) uint64 {
	var power float64
	if metricIndex >= 0 && sumMetricValues[metricIndex] > 0 {
		power = podMetricValue[metricIndex] / sumMetricValues[metricIndex] * float64(totalPower)
	} else {
		power = float64(totalPower) / podNumber
	}
	return uint64(math.Ceil(power))
}

func GetPodPowerRatio(podMetricValues [][]float64, otherNodePower uint64, nodeComponentPower source.RAPLPower, sumMetricValues []float64) (componentPowers []source.RAPLPower, otherPodPowers []uint64) {
	podNumber := len(podMetricValues)
	componentPowers = make([]source.RAPLPower, podNumber)
	otherPodPowers = make([]uint64, podNumber)
	podNumberDivision := float64(podNumber)

	// Package (PKG) domain measures the energy consumption of the entire socket, including the consumption of all the cores, integrated graphics and
	// also the "unknown" components such as last level caches and memory controllers
	pkgUnknownValue := nodeComponentPower.Pkg - nodeComponentPower.Core - nodeComponentPower.Uncore

	// find ratio power
	for index, podMetricValue := range podMetricValues {
		coreValue := getRatio(podMetricValue, coreMetricIndex, nodeComponentPower.Core, podNumberDivision, sumMetricValues)
		uncoreValue := getRatio(podMetricValue, uncoreMetricIndex, nodeComponentPower.Uncore, podNumberDivision, sumMetricValues)
		unknownValue := getRatio(podMetricValue, generalMetricIndex, pkgUnknownValue, podNumberDivision, sumMetricValues)
		dramValue := getRatio(podMetricValue, dramMetricIndex, nodeComponentPower.DRAM, podNumberDivision, sumMetricValues)
		otherValue := getRatio(podMetricValue, generalMetricIndex, otherNodePower, podNumberDivision, sumMetricValues)
		pkgValue := coreValue + uncoreValue + unknownValue
		componentPowers[index] = source.RAPLPower{
			Pkg:    pkgValue,
			Core:   coreValue,
			Uncore: uncoreValue,
			DRAM:   dramValue,
		}
		otherPodPowers[index] = otherValue
	}
	return
}
