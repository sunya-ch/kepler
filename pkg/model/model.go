/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package model

import (
	"github.com/sustainable-computing-io/kepler/pkg/power/rapl/source"
	"k8s.io/klog/v2"
)

var (
	ModelOutputTypeConverter = []string{
		"AbsPower", "AbsModelWeight", "AbsComponentPower", "AbsComponentModelWeight", "DynPower", "DynModelWeight", "DynComponentPower", "DynComponentModelWeight",
	}

	NodeTotalPowerModelValid, NodeComponentPowerModelValid, PodTotalPowerModelValid, PodComponentPowerModelValid bool
	EstimatorSidecarSocket                                                                                       = "/tmp/estimator.sock"

	/////////////////////////////////////////////////////
	// TODO: be configured by config package
	modelServerEndpoint = "http://kepler-model-server.monitoring.cluster.local:8100/model"
	// cgroupOnly
	dynCompURL = "https://raw.githubusercontent.com/sustainable-computing-io/kepler-model-server/main/tests/test_models/DynComponentModelWeight/CgroupOnly/ScikitMixed/ScikitMixed.json"

	NodeTotalPowerModelConfig     = ModelConfig{UseEstimatorSidecar: false}
	NodeComponentPowerModelConfig = ModelConfig{UseEstimatorSidecar: false}
	PodTotalPowerModelConfig      = ModelConfig{UseEstimatorSidecar: false}
	PodComponentPowerModelConfig  = ModelConfig{UseEstimatorSidecar: false, InitModelURL: dynCompURL}

	NodeTotalPowerModelFunc, PodTotalPowerModelFunc         func([][]float64, []string) ([]float64, error)
	NodeComponentPowerModelFunc, PodComponentPowerModelFunc func([][]float64, []string) (map[string][]float64, error)
	
	/////////////////////////////////////////////////////
)

// InitEstimateFunctions checks validity of power model and set estimate functions
func InitEstimateFunctions(usageMetrics, systemFeatures, systemValues []string) {
	var estimateFunc interface{}
	// init func for NodeTotalPower
	NodeTotalPowerModelValid, estimateFunc = initEstimateFunction(NodeTotalPowerModelConfig, AbsPower, AbsModelWeight, usageMetrics, systemFeatures, systemValues, true)
	if NodeTotalPowerModelValid {
		NodeTotalPowerModelFunc = estimateFunc.(func([][]float64, []string) ([]float64, error))
	}
	// init func for NodeComponentPower
	NodeComponentPowerModelValid, estimateFunc = initEstimateFunction(NodeComponentPowerModelConfig, AbsComponentPower, AbsComponentModelWeight, usageMetrics, systemFeatures, systemValues, false)
	if NodeComponentPowerModelValid {
		NodeComponentPowerModelFunc = estimateFunc.(func([][]float64, []string) (map[string][]float64, error))
	}
	// init func for PodTotalPower
	PodTotalPowerModelValid, estimateFunc = initEstimateFunction(PodTotalPowerModelConfig, DynPower, DynModelWeight, usageMetrics, systemFeatures, systemValues, true)
	if PodTotalPowerModelValid {
		PodTotalPowerModelFunc = estimateFunc.(func([][]float64, []string) ([]float64, error))
	}
	// init func for PodComponentPower
	PodComponentPowerModelValid, estimateFunc = initEstimateFunction(PodComponentPowerModelConfig, DynComponentPower, DynComponentModelWeight, usageMetrics, systemFeatures, systemValues, false)
	if PodComponentPowerModelValid {
		PodComponentPowerModelFunc = estimateFunc.(func([][]float64, []string) (map[string][]float64, error))
	}
}

// initEstimateFunction called by InitEstimateFunctions for each power model
func initEstimateFunction(modelConfig ModelConfig, archiveType, modelWeightType ModelOutputType, usageMetrics, systemFeatures, systemValues []string, isTotalPower bool) (valid bool, estimateFunc interface{}) {
	if modelConfig.UseEstimatorSidecar {
		// try init EstimatorSidecarConnector
		c := EstimatorSidecarConnector{
			Socket:         EstimatorSidecarSocket,
			UsageMetrics:   usageMetrics,
			OutputType:     archiveType,
			SystemFeatures: systemFeatures,
			ModelName:      modelConfig.SelectedModel,
			SelectFilter:   modelConfig.SelectFilter,
		}
		valid = c.Init(systemValues)
		if valid {
			if isTotalPower {
				estimateFunc = c.GetTotalPower
			} else {
				estimateFunc = c.GetComponentPower
			}
			return
		}
	}
	// set UseEstimatorSidecar to false as cannot init valid EstimatorSidecarConnector
	modelConfig.UseEstimatorSidecar = false
	r := LinearRegressor{
		Endpoint:       modelServerEndpoint,
		UsageMetrics:   usageMetrics,
		OutputType:     modelWeightType,
		SystemFeatures: systemFeatures,
		ModelName:      modelConfig.SelectedModel,
		SelectFilter:   modelConfig.SelectFilter,
		InitModelURL:   modelConfig.InitModelURL,
	}
	valid = r.Init()
	if isTotalPower {
		estimateFunc = r.GetTotalPower
	} else {
		estimateFunc = r.GetComponentPower
	}
	return valid, estimateFunc
}

func GetNodeTotalPower(usageValues []float64, systemValues []string) (valid bool, value uint64) {
	valid = false
	value = 0
	if NodeTotalPowerModelValid {
		powers, err := NodeTotalPowerModelFunc([][]float64{usageValues}, systemValues)
		if err != nil || len(powers) == 0 {
			return
		}
		valid = true
		value = uint64(powers[0])
		return
	}
	return
}

func GetNodeComponentPowers(usageValues []float64, systemValues []string) (valid bool, results source.RAPLPower) {
	results = source.RAPLPower{}
	valid = false
	if NodeComponentPowerModelValid {
		powers, err := NodeComponentPowerModelFunc([][]float64{usageValues}, systemValues)
		if err != nil {
			return
		}
		pkgPower := getComponentPower(powers, "pkg", 0)
		corePower := getComponentPower(powers, "core", 0)
		uncorePower := getComponentPower(powers, "uncore", 0)
		dramPower := getComponentPower(powers, "dram", 0)
		valid = true
		results = fillRAPLPower(pkgPower, corePower, uncorePower, dramPower)
		return
	}
	return
}

func GetPodPowers(usageValues [][]float64, systemValues []string, nodeTotalPower, totalGPUPower uint64, nodeComponentPower source.RAPLPower) (componentPodPowers []source.RAPLPower, otherPodPowers []uint64) {
	if nodeComponentPower.Pkg > 0 {
		if nodeTotalPower < nodeComponentPower.Pkg+nodeComponentPower.DRAM+totalGPUPower {
			// case: NodeTotalPower is invalid but NodeComponentPower model is available, set = pkg+DRAM+GPU
			nodeTotalPower = nodeComponentPower.Pkg + nodeComponentPower.DRAM + totalGPUPower
		}
	} else if nodeTotalPower > 0 {
		// case: no NodeComponentPower model but NodeTotalPower model is available, set = total-GPU, DRAM=0
		socketPower := nodeTotalPower - totalGPUPower
		nodeComponentPower = source.RAPLPower{
			Pkg:  socketPower,
			Core: socketPower,
		}
	}

	if nodeTotalPower > 0 {
		// total power all set, use ratio
		sumMetricValues := GetSumMetricValues(usageValues)
		nodeOtherPower := nodeTotalPower - nodeComponentPower.Pkg - nodeComponentPower.DRAM - totalGPUPower
		componentPodPowers, otherPodPowers = GetPodPowerRatio(usageValues, nodeOtherPower, nodeComponentPower, sumMetricValues)
	} else {
		// otherwise, use trained power model
		totalPowerValid, totalPodPowers := getPodTotalPower(usageValues, systemValues)
		var valid bool
		valid, componentPodPowers = getPodComponentPowers(usageValues, systemValues)
		if !valid {
			klog.V(5).Infoln("No PodComponentPower Model")
			return
		}
		otherPodPowers = make([]uint64, len(componentPodPowers))
		if totalPowerValid {
			for index, componentPower := range componentPodPowers {
				// TODO: include GPU into consideration
				otherPodPowers[index] = uint64(totalPodPowers[index]) - componentPower.Pkg - componentPower.DRAM
			}
		}
	}
	return componentPodPowers, otherPodPowers
}

func getPodTotalPower(usageValues [][]float64, systemValues []string) (valid bool, results []float64) {
	valid = false
	if PodTotalPowerModelValid {
		powers, err := PodTotalPowerModelFunc(usageValues, systemValues)
		if err != nil || len(powers) == 0 {
			return
		}
		results = powers
		valid = true
		return
	}
	return
}

func getPodComponentPowers(usageValues [][]float64, systemValues []string) (bool, []source.RAPLPower) {
	if PodComponentPowerModelValid {
		powers, err := PodComponentPowerModelFunc(usageValues, systemValues)
		if err != nil {
			return false, []source.RAPLPower{}
		}
		podNumber := len(usageValues)
		raplPowers := make([]source.RAPLPower, podNumber)
		for index := 0; index < podNumber; index++ {
			pkgPower := getComponentPower(powers, "pkg", index)
			corePower := getComponentPower(powers, "core", index)
			uncorePower := getComponentPower(powers, "uncore", index)
			dramPower := getComponentPower(powers, "dram", index)
			raplPowers[index] = fillRAPLPower(pkgPower, corePower, uncorePower, dramPower)
		}
		return true, raplPowers
	}
	return false, []source.RAPLPower{}
}

// getComponentPower checks if component key is present in powers response and fills with single 0
func getComponentPower(powers map[string][]float64, componentKey string, index int) uint64 {
	values := powers[componentKey]
	if index >= len(values) {
		return 0
	} else {
		return uint64(values[index])
	}
}

func fillRAPLPower(pkgPower, corePower, uncorePower, dramPower uint64) source.RAPLPower {
	if pkgPower < corePower+uncorePower {
		pkgPower = corePower + uncorePower
	}
	if corePower == 0 {
		corePower = pkgPower - uncorePower
	}
	return source.RAPLPower{
		Core:   corePower,
		Uncore: uncorePower,
		DRAM:   dramPower,
		Pkg:    pkgPower,
	}
}
