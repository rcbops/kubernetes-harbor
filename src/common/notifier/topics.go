package notifier

import (
	"github.com/rcbops/kubernetes-harbor/src/common"
)

//Define global topic names
const (
	//ScanAllPolicyTopic is for notifying the change of scanning all policy.
	ScanAllPolicyTopic = common.ScanAllPolicy
)
