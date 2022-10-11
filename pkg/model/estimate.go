package model

import (
	"encoding/json"
	"fmt"
	"net"

	"k8s.io/klog/v2"
)

type PowerRequest struct {
	UsageMetrics   []string    `json:"metrics"`
	UsageValues    [][]float64 `json:"values"`
	OutputType     string      `json:"output_type"`
	SystemFeatures []string    `json:"system_features"`
	SystemValues   []string    `json:"system_values"`
	ModelName      string      `json:"model_name"`
	SelectFilter   string      `json:"filter"`
}

type TotalPowerResponse struct {
	Powers  []float64   `json:"powers"`
	Message string      `json:"msg"`
}

type ComponentPowerResponse struct {
	Powers  map[string][]float64   `json:"powers"`
	Message string      `json:"msg"`
}



type EstimatorSidecarConnector struct {
	Socket         string
	UsageMetrics   []string
	OutputType     ModelOutputType
	SystemFeatures []string
	ModelName      string
	SelectFilter   string
	valid          bool
	isComponent    bool
}

func (c *EstimatorSidecarConnector) Init(systemValues []string) bool {
	zeros := make([]float64, len(c.UsageMetrics))
	usageValues := [][]float64{zeros}
	c.isComponent = isComponentType(c.OutputType)
	_, err := c.makeRequest(usageValues, systemValues)
	if err == nil {
		c.valid = true
	} else {
		c.valid = false
	}
	return c.valid
}

func (c *EstimatorSidecarConnector) makeRequest(usageValues [][]float64, systemValues []string) (interface{}, error) {
	powerRequest := PowerRequest{
		ModelName:      c.ModelName,
		UsageMetrics:   c.UsageMetrics,
		UsageValues:    usageValues,
		OutputType:     c.OutputType.String(),
		SystemFeatures: c.SystemFeatures,
		SystemValues:   systemValues,
		SelectFilter:   c.SelectFilter,
	}
	powerRequestJSON, err := json.Marshal(powerRequest)
	if err != nil {
		klog.V(4).Infof("marshal error: %v (%v)", err, powerRequest)
		return nil, err
	}

	conn, err := net.Dial("unix", c.Socket)
	if err != nil {
		klog.V(4).Infof("dial error: %v", err)
		return nil, err
	}
	defer conn.Close()
	
	_, err = conn.Write(powerRequestJSON)
	
	if err != nil {
		klog.V(4).Infof("estimator write error: %v", err)
		return nil, err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		klog.V(4).Infof("estimator read error: %v", err)
		return nil, err
	}
	var powers interface{}
	if c.isComponent {
		var powerResponse ComponentPowerResponse
		err = json.Unmarshal(buf[0:n], &powerResponse)
		powers = powerResponse.Powers
	} else {
		var powerResponse TotalPowerResponse
		err = json.Unmarshal(buf[0:n], &powerResponse)
		powers = powerResponse.Powers
	}
	if err != nil {
		klog.V(4).Info("estimator unmarshal error: %v (%s)", err, string(buf[0:n]))
		return nil, err
	}
	return powers, nil
}

func (c *EstimatorSidecarConnector) GetTotalPower(usageValues [][]float64, systemValues []string) ([]float64, error) {
	if !c.valid {
		return []float64{}, fmt.Errorf("invalid power model call: %s", c.OutputType.String())
	}
	powers, err := c.makeRequest(usageValues, systemValues)
	if err != nil {
		return []float64{}, err
	}
	return powers.([]float64), err
}

func (c *EstimatorSidecarConnector) GetComponentPower(usageValues [][]float64, systemValues []string) (map[string][]float64, error) {
	if !c.valid {
		return map[string][]float64{}, fmt.Errorf("invalid power model call: %s", c.OutputType.String())
	}
	powers, err := c.makeRequest(usageValues, systemValues)
	if err != nil {
		return map[string][]float64{}, err
	}
	return powers.(map[string][]float64), err
}
