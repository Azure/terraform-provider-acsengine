package acsengine

import (
	"fmt"
	"math"
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
)

// validateIntInSlice returns a SchemaValidateFunc which tests if the provided value
// is of type int and matches the value of an element in the valid slice
func validateIntInSlice(valid []int) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
		v, ok := i.(int)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be int", k))
			return
		}

		for _, str := range valid {
			if v == str {
				return
			}
		}

		es = append(es, fmt.Errorf("expected %q to be one of %v, got %v", k, valid, v))
		return
	}
}

// intBetweenDivisibleBy returns a SchemaValidateFunc which tests if the provided value
// is of type int and is between min and max (inclusive) and is divisible by a given number
func validateIntBetweenDivisibleBy(min, max, divisor int) schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
		v, ok := i.(int)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be int", k))
			return
		}

		if v < min || v > max {
			es = append(es, fmt.Errorf("expected %s to be in the range (%d - %d), got %d", k, min, max, v))
			return
		}

		if math.Mod(float64(v), float64(divisor)) != 0 {
			es = append(es, fmt.Errorf("expected %s to be divisible by %d", k, divisor))
			return
		}

		return
	}
}

func validateCollation() schema.SchemaValidateFunc {
	return func(i interface{}, k string) (s []string, es []error) {
		v, ok := i.(string)
		if !ok {
			es = append(es, fmt.Errorf("expected type of %s to be string", k))
			return
		}

		matched, _ := regexp.MatchString(`^[A-Za-z0-9_. ]+$`, v)

		if !matched {
			es = append(es, fmt.Errorf("%s contains invalid characters, only underscores are supported, got %s", k, v))
			return
		}

		return
	}
}
