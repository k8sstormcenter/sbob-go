package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

type ExtensionsConflictError struct {
	Key string
}

func (e *ExtensionsConflictError) Error() string {
	return fmt.Sprintf("extensions key %q has conflicting values", e.Key)
}

func MergeExtensions(a, b json.RawMessage) (json.RawMessage, error) {
	if len(a) == 0 {
		return cloneRaw(b), nil
	}
	if len(b) == 0 {
		return cloneRaw(a), nil
	}
	ca, err := compactRaw(a)
	if err != nil {
		return nil, err
	}
	cb, err := compactRaw(b)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(ca, cb) {
		return cloneRaw(a), nil
	}
	var ma, mb map[string]json.RawMessage
	if err := json.Unmarshal(ca, &ma); err != nil {
		return nil, fmt.Errorf("extensions is not a JSON object: %w", err)
	}
	if err := json.Unmarshal(cb, &mb); err != nil {
		return nil, fmt.Errorf("extensions is not a JSON object: %w", err)
	}
	keys := make([]string, 0, len(mb))
	for k := range mb {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vb := mb[k]
		va, ok := ma[k]
		if !ok {
			ma[k] = vb
			continue
		}
		xa, err := compactRaw(va)
		if err != nil {
			return nil, err
		}
		xb, err := compactRaw(vb)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(xa, xb) {
			return nil, &ExtensionsConflictError{Key: k}
		}
	}
	return json.Marshal(ma)
}

func compactRaw(v json.RawMessage) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, v); err != nil {
		return nil, fmt.Errorf("extensions is not valid JSON: %w", err)
	}
	return buf.Bytes(), nil
}
