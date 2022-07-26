package graphqltransportws

type Context map[string]interface{}

// ConnectionInitReceived gets connection recieved value
func (c Context) ConnectionInitReceived() bool {
	v, ok := c["connectionInitReceived"]
	if ok {
		received, ok := v.(bool)
		if ok {
			return received
		}
	}
	return false
}

// Acknowledged gets acknowledged value
func (c Context) Acknowledged() bool {
	v, ok := c["acknowledged"]
	if ok {
		acknowledged, ok := v.(bool)
		if ok {
			return acknowledged
		}
	}
	return false
}

// ConnectionParams gets connection params
func (c Context) ConnectionParams() map[string]interface{} {
	v, ok := c["connectionParams"]
	if ok {
		connectionParams, ok := v.(map[string]interface{})
		if ok {
			return connectionParams
		}
	}

	return nil
}
