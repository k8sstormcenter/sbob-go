package dynamicpathdetector

import "encoding/json"

type Exec struct {
	Path         string
	Args         []string
	Envs         []string
	ArgsRequired bool
}

type Open struct {
	Path  string
	Flags []string
}

const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
)

type Endpoint struct {
	Endpoint  string
	Methods   []string
	Internal  bool
	Direction string
	Headers   []byte
}

func (e *Endpoint) GetHeaders() (map[string][]string, error) {
	headers := make(map[string][]string)
	if err := json.Unmarshal(e.Headers, &headers); err != nil {
		return nil, err
	}
	return headers, nil
}
