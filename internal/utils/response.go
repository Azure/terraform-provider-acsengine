package utils

import (
	"net/http"

	"github.com/Azure/go-autorest/autorest"
)

// ResponseWasNotFound checks if HTTP response was a status "not found"
func ResponseWasNotFound(resp autorest.Response) bool {
	return responseWasStatusCode(resp, http.StatusNotFound)
}

func responseWasStatusCode(resp autorest.Response, statusCode int) bool {
	if r := resp.Response; r != nil {
		if r.StatusCode == statusCode {
			return true
		}
	}

	return false
}
