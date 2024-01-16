/*
Copyright 2019 The Kubernetes Authors.

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

package aiplugin

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	"net/http"
	"strings"
)

type AiSchedulerPlugin struct {
	aiClient *http.Client
	handle   framework.FrameworkHandle
}

type InstanceInfo struct {
	Env    string   `json:"env"`
	App    string   `json:"app"`
	AppId  string   `json:"appId"`
	Ip     string   `json:"ip"`
	NodeIp []string `json:"nodeIp"`
}
type InstanceAllocationRequest struct {
	Ins_info InstanceInfo `json:"ins_info"`
	Env      string       `json:"env"`
}

type NodeScore struct {
	Score int    `json:"score"`
	Ip    string `json:"ip"`
}

type InstanceAllocationData struct {
	App        string      `json:"app"`
	Env        string      `json:"env"`
	AppId      string      `json:"appId"`
	Ip         string      `json:"ip"`
	NodeScores []NodeScore `json:"nodeScores"`
}

type InstanceAllocationResp struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    InstanceAllocationData `json:"data"`
}

var _ framework.ScorePlugin = &AiSchedulerPlugin{}
var _ framework.PreScorePlugin = &AiSchedulerPlugin{}

// Name is the name of the plugin used in Registry and configurations.
const (
	Name             = "AiPlugin"
	preScoreStateKey = "PreScore" + Name
)

// Name returns name of the plugin. It is used in logs, etc.
func (mc AiSchedulerPlugin) Name() string {
	return Name
}

type stateData struct {
	data InstanceAllocationResp
}

func (s *stateData) Clone() framework.StateData {
	copy := &stateData{
		data: s.data,
	}
	return copy
}

// PreScore builds and writes cycle state used by Score and NormalizeScore.
func (as *AiSchedulerPlugin) PreScore(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodes []*v1.Node) *framework.Status {
	if len(nodes) == 0 {
		// No nodes to score.
		return nil
	}

	podLabels := pod.Labels
	if podLabels == nil {
		podLabels = map[string]string{}
	}
	//aiScheduler := podLabels["aiScheduler"]
	//if aiScheduler != "aiScheduler" {
	//	klog.V(1).Infof("ai PreScore isnot aiScheduler")
	//	return nil
	//}
	var instanceInfo InstanceInfo
	instanceInfo.App = pod.Labels["app"]
	instanceInfo.AppId = pod.Labels["appid"]
	instanceInfo.Env = pod.Labels["env"]
	instanceInfo.Ip = pod.Labels["ip"]

	nodeNames := make([]string, len(nodes))
	for i, node := range nodes {
		nodeNames[i] = node.Name
	}
	instanceInfo.NodeIp = nodeNames

	var instanceAllocationRequest InstanceAllocationRequest
	instanceAllocationRequest.Env = "TEST"
	instanceAllocationRequest.Ins_info = instanceInfo
	instanceAllocationRequestJson, _ := json.Marshal(instanceAllocationRequest)

	request, err := http.NewRequest("POST", "http://fat-wdkapp.ppdapi.com/instance_allocation_online", strings.NewReader(string(instanceAllocationRequestJson)))
	if err != nil {
		klog.V(3).Infof("ai PreScore http.NewRequest: %v", err)
		return nil
	}
	request.Header.Set("Content-Type", "application/json")
	resp, err := as.aiClient.Do(request)
	if err != nil {
		klog.V(3).Infof("ai PreScore as.aiClient.Do wdkapp.ppdapi.com error: %v", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 100000 {
		return nil
	}
	var result InstanceAllocationResp
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		klog.V(3).Infof("ai PreScore json.NewDecoder : %v", err)
		return nil
	}
	if result.Code != 100000 {
		klog.V(3).Infof("ai PreScore as.aiClient.Do result.Code != 100000 : %v", err)
		return nil
	}
	klog.V(1).Infof("ai PreScore result : %v", result)
	cycleState.Write(preScoreStateKey, &stateData{data: result})
	return nil
}

func (as *AiSchedulerPlugin) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	nodeInfo, err := as.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("getting node %q from Snapshot: %v", nodeName, err))
	}

	node := nodeInfo.Node()
	if node == nil {
		return 0, framework.NewStatus(framework.Error, "node not found")
	}

	podLabels := pod.Labels
	if podLabels == nil {
		podLabels = map[string]string{}
	}
	return int64(0), nil
}

// ScoreExtensions of the Score plugin.
func (as *AiSchedulerPlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// New initializes a new plugin and returns it.
func New(_ *runtime.Unknown, _ framework.FrameworkHandle) (framework.Plugin, error) {
	return &AiSchedulerPlugin{}, nil
}
