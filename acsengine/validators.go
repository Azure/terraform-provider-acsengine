package acsengine

import (
	"fmt"
)

func validateMasterProfileCount(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	capacities := map[int]bool{
		1: true,
		3: true,
		5: true,
	}

	if !capacities[value] {
		errors = append(errors, fmt.Errorf("the number of master nodes must be 1, 3 or 5"))
	}
	return
}

func validateAgentPoolProfileCount(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	if value > 100 || value <= 0 {
		errors = append(errors, fmt.Errorf("the count for an agent pool profile can only be between 1 and 100"))
	}
	return
}
