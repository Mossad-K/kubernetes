package plugins

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
)

// StatelessPreBindExample is an example of a simple plugin that has no state
// and implements only one hook for prebind.
type CommunicatingPlugin struct{}

// var _ framework.PreBindPlugin = CommunicatingPlugin{}
var _ framework.ScorePlugin = CommunicatingPlugin{}

// 插件名称
const Name = "sample-plugin"

// Name returns name of the plugin. It is used in logs, etc.
func (cp CommunicatingPlugin) Name() string {
	return Name
}

// PreBind is the functions invoked by the framework at "prebind" extension point.
//func (sr CommunicatingPlugin) PreBind(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
//	if pod == nil {
//		return framework.NewStatus(framework.Error, fmt.Sprintf("pod cannot be nil"))
//	}
//	klog.V(3).Infof("PreBind pod: %v, node: %v", pod.Name, nodeName)
//
//	return nil
//}

// Score invoked at the score extension point.
func (cp CommunicatingPlugin) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	if pod == nil {
		return int64(0), framework.NewStatus(framework.Error, fmt.Sprintf("pod cannot be nil"))
	}
	klog.V(3).Infof("PreBind pod: %v, node: %v", pod.Name, nodeName)
	return int64(0), nil
}

func (cp CommunicatingPlugin) ScoreExtensions() framework.ScoreExtensions {
	klog.V(3).Infof("ScoreExtensions")
	return nil
}

// New initializes a new plugin and returns it.
func New(_ *runtime.Unknown, _ framework.FrameworkHandle) (framework.Plugin, error) {
	return &CommunicatingPlugin{}, nil
}
