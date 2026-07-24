package agentclient

import (
	"crypto/tls"
	"net/http"
	"time"
)

// tokenAuthRoundTripper is a custom http.RoundTripper that injects
// an Authorization header into every request.
type tokenAuthRoundTripper struct {
	token string
	rt    http.RoundTripper
}

func (t *tokenAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	reqClone := req.Clone(req.Context())
	if t.token != "" {
		reqClone.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.rt.RoundTrip(reqClone)
}

// NewClient returns an *http.Client configured to include the provided token
// as a Bearer token in the Authorization header.
func NewClient(token string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &tokenAuthRoundTripper{
			token: token,
			rt:    transport,
		},
	}
}

// NewPruneClient returns an *http.Client configured for longer timeout operations
// like pruning.
func NewPruneClient(token string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return &http.Client{
		Timeout: 120 * time.Second,
		Transport: &tokenAuthRoundTripper{
			token: token,
			rt:    transport,
		},
	}
}

// NewStreamingClient returns an *http.Client configured for streaming operations
// with no timeout.
func NewStreamingClient(token string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return &http.Client{
		Transport: &tokenAuthRoundTripper{
			token: token,
			rt:    transport,
		},
	}
}
