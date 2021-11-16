package conflux

type Client struct {
	// p  *params.ChainConfig
	// tc *tracers.TraceConfig

	c JSONRPC
	// g GraphQL

	// traceSemaphore *semaphore.Weighted

	// skipAdminCalls bool
}

func NewClient(url string) (*Client, error) {
	return nil, nil
}

// Close shuts down the RPC client connection.
func (ec *Client) Close() {
	ec.c.Close()
}
