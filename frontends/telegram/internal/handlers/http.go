package handlers

import (
	"net/http"
)

// httpGet is a package-level variable for making HTTP GET requests (for easy testing).
var httpGet = func(url string) (*http.Response, error) {
	return http.Get(url) //nolint:noctx
}
