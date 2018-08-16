package acsengine

import (
	"testing"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/operations"
	"github.com/stretchr/testify/assert"
)

func TestSetCountForTemplate(t *testing.T) {
	cases := []struct {
		DesiredAgentCount int
		HighestUsedIndex  int
		CurrentNodeCount  int
		Expected          int
	}{
		{
			DesiredAgentCount: 2,
			HighestUsedIndex:  0,
			CurrentNodeCount:  1,
			Expected:          2,
		},
		{
			DesiredAgentCount: 2,
			HighestUsedIndex:  1,
			CurrentNodeCount:  1,
			Expected:          3,
		},
	}

	for _, tc := range cases {
		sc := operations.ScaleClient{
			DesiredAgentCount: tc.DesiredAgentCount,
		}
		countForTemplate := setCountForTemplate(&sc, tc.HighestUsedIndex, tc.CurrentNodeCount)
		assert.Equal(t, tc.Expected, countForTemplate, "count for template should be the same")
	}
}

func TestSetWindowsIndex(t *testing.T) {
	cases := []struct {
		WindowsIndex  int
		AgentPoolName string
	}{
		{
			WindowsIndex:  1,
			AgentPoolName: "agentpool1",
		},
		{
			WindowsIndex:  2,
			AgentPoolName: "agentpool2",
		},
	}

	templateJSON := map[string]interface{}{
		"variables": map[string]interface{}{},
	}

	for _, tc := range cases {
		sc := operations.ScaleClient{
			AgentPool: &api.AgentPoolProfile{
				Name: tc.AgentPoolName,
			},
		}
		setWindowsIndex(&sc, tc.WindowsIndex, templateJSON)

		assert.Equal(t, tc.WindowsIndex, templateJSON["variables"].(map[string]interface{})[sc.AgentPool.Name+"Index"], "Windows index should be the same")
	}
}

func TestVMsToDeleteList(t *testing.T) {
	cases := []struct {
		DesiredNodeCount int
	}{
		{
			DesiredNodeCount: 2,
		},
		{
			DesiredNodeCount: 1,
		},
	}

	for _, tc := range cases {
		vms := []string{
			"agentpool1vm0",
			"agentpool1vm1",
			"agentpool1vm3",
		}

		vmsToDelete := vmsToDeleteList(vms, len(vms), tc.DesiredNodeCount)

		assert.Equal(t, len(vms)-tc.DesiredNodeCount, len(vmsToDelete), "number of VMs to delete is incorrect")
		assert.Equal(t, vms[2], vmsToDelete[0], "first VM to delete should be last vm in original slice")
	}
}
