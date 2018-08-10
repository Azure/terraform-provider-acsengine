package acsengine

import (
	"testing"

	"github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/terraform-provider-acsengine/acsengine/helpers/client"
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
		sc := client.ScaleClient{
			DesiredAgentCount: tc.DesiredAgentCount,
		}
		countForTemplate := setCountForTemplate(&sc, tc.HighestUsedIndex, tc.CurrentNodeCount)
		assert.Equal(t, countForTemplate, tc.Expected, "count for template should be the same")
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
		sc := client.ScaleClient{
			AgentPool: &api.AgentPoolProfile{
				Name: tc.AgentPoolName,
			},
		}
		setWindowsIndex(&sc, tc.WindowsIndex, templateJSON)

		assert.Equal(t, templateJSON["variables"].(map[string]interface{})[sc.AgentPool.Name+"Index"], tc.WindowsIndex, "Windows index should be the same")
	}
}
