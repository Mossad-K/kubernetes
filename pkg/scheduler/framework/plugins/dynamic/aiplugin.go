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

package multipoint

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	"net/http"
	"strings"
)

type AiSchedulerPlugin struct {
	aiClient *http.Client
	handle   framework.FrameworkHandle
}
type InstanceInfo struct {
	env    string
	app    string
	appId  string
	ip     string
	nodeId []string
}
type InstanceAllocationRequest struct {
	ins_info InstanceInfo
	env      string
}

type NodeScore struct {
	score int
	ip    string
}

type InstanceAllocationData struct {
	app        string
	env        string
	appId      string
	ip         string
	nodeScores []NodeScore
}

type InstanceAllocationResp struct {
	code    int
	message string
	data    InstanceAllocationData
}

var _ framework.ScorePlugin = AiSchedulerPlugin{}
var _ framework.PreScorePlugin = AiSchedulerPlugin{}

// Name is the name of the plugin used in Registry and configurations.
const Name = "ai-plugin"

// Name returns name of the plugin. It is used in logs, etc.
func (mc AiSchedulerPlugin) Name() string {
	return Name
}

type stateData struct {
	data string
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
	var instanceInfo InstanceInfo
	instanceInfo.app = pod.Labels["app"]
	instanceInfo.appId = pod.Labels["appid"]
	instanceInfo.env = pod.Labels["env"]
	instanceInfo.ip = pod.Labels["ip"]

	nodeNames := make([]string, len(nodes))
	for i, node := range nodes {
		nodeNames[i] = node.Name
	}
	instanceInfo.nodeId = nodeNames

	var instanceAllocationRequest InstanceAllocationRequest
	instanceAllocationRequest.env = "TEST"
	instanceAllocationRequest.ins_info = instanceInfo
	instanceAllocationRequestJson, _ := json.Marshal(instanceAllocationRequest)

	request, err := http.NewRequest("POST", "http://fat-wdkapp.ppdapi.com/instance_allocation_online", strings.NewReader(string(instanceAllocationRequestJson)))
	if err != nil {
		return nil
	}
	resp, err := as.aiClient.Do(request)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 100000 {
		return nil
	}
	var result InstanceAllocationResp
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil
	}
	if result.code != 100000 {

	}
	//cycleState.Write(preScoreStateKey, state)
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

	nodeLabels := node.Labels
	if nodeLabels == nil {
		nodeLabels = map[string]string{}
	}
	return int64(0), nil
}

// ScoreExtensions of the Score plugin.
func (as *AiSchedulerPlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// Score returns the score of scheduling a pod on a specific node.
func (as *AiSchedulerPlugin) NormalizeScore(ctx context.Context, cycleState *framework.CycleState, p *v1.Pod, scores framework.NodeScoreList) *framework.Status {

	return nil
}

// New initializes a new plugin and returns it.
func New(_ *runtime.Unknown, _ framework.FrameworkHandle) (framework.Plugin, error) {
	return &AiSchedulerPlugin{}, nil
}
