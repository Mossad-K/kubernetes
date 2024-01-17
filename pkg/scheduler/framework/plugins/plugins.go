package plugins

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

// StatelessPreBindExample is an example of a simple plugin that has no state
// and implements only one hook for prebind.

// var _ framework.PreBindPlugin = CommunicatingPlugin{}
var _ framework.PreScorePlugin = CommunicatingPlugin{}

// 插件名称
const Name = "sample-plugin"
const preScoreStateKey = "PreScore" + Name

// Name returns name of the plugin. It is used in logs, etc.
func (cp CommunicatingPlugin) Name() string {
	return Name
}

func (cp CommunicatingPlugin) PreScore(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodes []*v1.Node) *framework.Status {
	if pod == nil {
		return framework.NewStatus(framework.Error, fmt.Sprintf("pod cannot be nil"))
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

	klog.V(3).Infof("ai PreScore app: %v appId: %v env: %v ip: %v", instanceInfo.App, instanceInfo.AppId, instanceInfo.Env, instanceInfo.Ip)

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
	client := http.DefaultClient
	klog.V(3).Infof("ai PreScore query fat-wdkapp.ppdapi.com app: %v appId: %v env: %v ip: %v", instanceInfo.App, instanceInfo.AppId, instanceInfo.Env, instanceInfo.Ip)
	resp, err := client.Do(request)
	klog.V(3).Infof("ai PreScore query done fat-wdkapp.ppdapi.com app: %v appId: %v env: %v ip: %v", instanceInfo.App, instanceInfo.AppId, instanceInfo.Env, instanceInfo.Ip)
	if err != nil {
		klog.V(3).Infof("ai PreScore as.aiClient.Do wdkapp.ppdapi.com error: %v", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		klog.V(3).Infof("ai PreScore lresp.StatusCode != 200 ")
		return nil
	}
	var result InstanceAllocationResp
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		klog.V(3).Infof("ai PreScore json.NewDecoder : %v", err)
		return nil
	}
	if result.Code != 100000 {
		klog.V(3).Infof("ai PreScore as.aiClient.Do result.Code != 100000")
		return nil
	}
	klog.V(3).Infof("ai PreScore result : %v", result)

	if len(result.Data.NodeScores) <= 0 {
		klog.V(3).Infof("ai PreScore len(result.Data.NodeScores) <= 0 ")
		return nil
	}

	aiScore := make(map[string]int64)

	for _, nodeScore := range result.Data.NodeScores {
		aiScore[nodeScore.Ip] = int64(nodeScore.Score)
	}
	klog.V(3).Infof("ai PreScore app: %v appId: %v env: %v ip: %v cycleState.Write : %v", instanceInfo.App, instanceInfo.AppId, instanceInfo.Env, instanceInfo.Ip, aiScore)

	aiPreScoreState := &AiPreScoreState{
		aiScore: aiScore,
	}
	cycleState.Write(preScoreStateKey, aiPreScoreState)
	return nil
}

// Score invoked at the score extension point.
func (cp CommunicatingPlugin) Score(ctx context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	nodeInfo, err := cp.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil || nodeInfo.Node() == nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("getting node %q from Snapshot: %v, node is nil: %v", nodeName, err, nodeInfo.Node() == nil))
	}
	//
	//podLabels := pod.Labels
	//if podLabels == nil {
	//	podLabels = map[string]string{}
	//}
	//aiScheduler := podLabels["aiScheduler"]
	//if aiScheduler != "aiScheduler" {
	//	klog.V(1).Infof("ai PreScore isnot aiScheduler")
	//	return int64(0), nil
	//}

	app := pod.Labels["app"]
	appId := pod.Labels["appid"]
	env := pod.Labels["env"]
	ip := pod.Labels["ip"]
	score, err := getAiPreScoreState(cycleState, nodeName)
	if err != nil {
		klog.V(3).Infof("ai Score error : %v", err)
		return int64(0), nil
	}
	klog.V(3).Infof("ai app: %v appId: %v env: %v ip: %v nodeIp: %v Score: %v", app, appId, env, ip, nodeName, score)
	return score, nil
}

func getAiPreScoreState(cycleState *framework.CycleState, nodeName string) (int64, error) {
	c, err := cycleState.Read(preScoreStateKey)
	if err != nil {
		return int64(0), fmt.Errorf("Error reading %q from cycleState: %v", preScoreStateKey, err)
	}

	aiPreScoreState, ok := c.(*AiPreScoreState)
	if !ok {
		return int64(0), fmt.Errorf("%+v  convert to interpodaffinity.preScoreState error", c)
	}
	return aiPreScoreState.aiScore[nodeName], nil
}

func (s *AiPreScoreState) Clone() framework.StateData {
	return s
}

// ScoreExtensions of the Score plugin.
func (as CommunicatingPlugin) ScoreExtensions() framework.ScoreExtensions {
	klog.V(3).Infof("ai ScoreExtensions")
	return nil
}

// New initializes a new plugin and returns it.
func New(configuration *runtime.Unknown, f framework.FrameworkHandle) (framework.Plugin, error) {
	args := &Args{}
	if err := framework.DecodeInto(configuration, args); err != nil {
		return nil, err
	}

	return &CommunicatingPlugin{
		args:   args,
		handle: f,
	}, nil
}

type CommunicatingPlugin struct {
	args   *Args
	handle framework.FrameworkHandle
}

type Args struct {
	FavoriteColor  string `json:"favorite_color,omitempty"`
	FavoriteNumber int    `json:"favorite_number,omitempty"`
	ThanksTo       string `json:"thanks_to,omitempty"`
}

// preScoreState computed at PreScore and used at Score.
type AiPreScoreState struct {
	aiScore map[string]int64
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
