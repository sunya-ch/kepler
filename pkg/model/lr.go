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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"

	"k8s.io/klog/v2"
)

type ModelRequest struct {
	ModelName    string   `json:"model_name"`
	MetricNames  []string `json:"metrics"`
	SelectFilter string   `json:"filter"`
	OutputType   string   `json:"output_type"`
}

type CategoricalFeature struct {
	Weight float64 `json:"weight"`
}

type NormalizedNumericalFeature struct {
	Mean     float64 `json:"mean"`
	Variance float64 `json:"variance"`
	Weight   float64 `json:"weight"`
}

type AllWeights struct {
	BiasWeight           float64                                  `json:"Bias_Weight"`
	CategoricalVariables map[string]map[string]CategoricalFeature `json:"Categorical_Variables"`
	NumericalVariables   map[string]NormalizedNumericalFeature    `json:"Numerical_Variables"`
}

type ModelWeights struct {
	AllWeights `json:"All_Weights"`
}

func (weights ModelWeights) getIndexedWeights(metricNames, systemFeatures []string) (categoricalWeights []map[string]CategoricalFeature, numericalWeights []NormalizedNumericalFeature) {
	w := weights.AllWeights
	for _, m := range systemFeatures {
		categoricalWeights = append(categoricalWeights, w.CategoricalVariables[m])
	}
	for _, m := range metricNames {
		numericalWeights = append(numericalWeights, w.NumericalVariables[m])
	}
	return
}

func (weights ModelWeights) predict(metricNames []string, values [][]float64, systemFeatures, systemValues []string) []float64 {
	categoricalWeights, numericalWeights := weights.getIndexedWeights(metricNames, systemFeatures)
	basePower := weights.AllWeights.BiasWeight
	for index, coeffMap := range categoricalWeights {
		basePower += coeffMap[systemValues[index]].Weight
	}
	var powers []float64
	for _, vals := range values {
		power := basePower
		for index, coeff := range numericalWeights {
			if coeff.Weight == 0 {
				continue
			}
			// Normalize each Numerical Feature's prediction given Keras calculated Mean and Variance.
			normalizedX := (vals[index] - coeff.Mean) / math.Sqrt(coeff.Variance)
			power += coeff.Weight * normalizedX
		}
		powers = append(powers, power)
	}
	return powers
}

type ComponentModelWeights map[string]ModelWeights

type LinearRegressor struct {
	Endpoint       string
	UsageMetrics   []string
	OutputType     ModelOutputType
	SystemFeatures []string
	ModelName      string
	SelectFilter   string
	InitModelURL   string
	valid          bool
	modelWeight    interface{}
}

func (r *LinearRegressor) Init() bool {
	// try getting weight from model server
	weight, err := r.getWeightFromServer()
	if err != nil && r.InitModelURL != "" {
		// next try loading from URL by config
		weight, err = r.loadWeightFromURL()
	}
	if err == nil {
		r.valid = true
		r.modelWeight = weight
		klog.V(3).Infof("LR Model (%s) Weight: %v", r.OutputType.String(), r.modelWeight)
	} else {
		r.valid = false
	}
	return r.valid
}

func (r *LinearRegressor) getWeightFromServer() (interface{}, error) {
	modelRequest := ModelRequest{
		ModelName:    r.ModelName,
		MetricNames:  append(r.UsageMetrics, r.SystemFeatures...),
		SelectFilter: r.SelectFilter,
		OutputType:   r.OutputType.String(),
	}
	modelRequestJSON, err := json.Marshal(modelRequest)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %v (%v)", err, modelRequest)
	}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, r.Endpoint, bytes.NewBuffer(modelRequestJSON))
	if err != nil {
		return nil, fmt.Errorf("connection error: %s (%v)", r.Endpoint, err)
	}

	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("connection error: %v (%v)", err, r.Endpoint)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status not ok: %v (%v)", response.Status, modelRequest)
	}
	body, _ := io.ReadAll(response.Body)

	if isComponentType(r.OutputType) {
		var response ComponentModelWeights
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, fmt.Errorf("model unmarshal error: %v (%s)", err, string(body))
		}
		return response, nil
	} else {
		var response ModelWeights
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, fmt.Errorf("model unmarshal error: %v (%s)", err, string(body))
		}
		return response, nil
	}
}

func (r *LinearRegressor) loadWeightFromURL() (interface{}, error) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, r.InitModelURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("connection error: %s (%v)", r.InitModelURL, err)
	}
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("connection error: %v (%v)", err, r.InitModelURL)
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if isComponentType(r.OutputType) {
		var content ComponentModelWeights
		err = json.Unmarshal(body, &content)
		if err != nil {
			return nil, fmt.Errorf("model unmarshal error: %v (%s)", err, string(body))
		}
		return content, nil
	} else {
		var content ModelWeights
		err = json.Unmarshal(body, &content)
		if err != nil {
			return nil, fmt.Errorf("model unmarshal error: %v (%s)", err, string(body))
		}
		return content, nil
	}
}

func (r *LinearRegressor) GetTotalPower(usageValues [][]float64, systemValues []string) ([]float64, error) {
	if !r.valid {
		return []float64{}, fmt.Errorf("invalid power model call: %s", r.OutputType.String())
	}
	return r.modelWeight.(ModelWeights).predict(r.UsageMetrics, usageValues, r.SystemFeatures, systemValues), nil
}

func (r *LinearRegressor) GetComponentPower(usageValues [][]float64, systemValues []string) (map[string][]float64, error) {
	if !r.valid {
		return map[string][]float64{}, fmt.Errorf("invalid power model call: %s", r.OutputType.String())
	}
	compPowers := make(map[string][]float64)
	for comp, weight := range r.modelWeight.(ComponentModelWeights) {
		compPowers[comp] = weight.predict(r.UsageMetrics, usageValues, r.SystemFeatures, systemValues)
	}
	return compPowers, nil
}
