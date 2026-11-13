package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const nullObject = ""

type objectReader struct {
	raw  map[string]json.RawMessage
	seen map[string]bool
	err  error
}

func readObject(data []byte) (*objectReader, error) {
	r := &objectReader{}
	if err := json.Unmarshal(data, &r.raw); err != nil {
		return nil, err
	}
	if isNull(data) {
		r.seen = map[string]bool{nullObject: true}
	}
	return r, nil
}

func isNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func readField[T any](r *objectReader, key string, dst *T) {
	raw, ok := r.raw[key]
	if !ok {
		return
	}
	delete(r.raw, key)
	if r.seen == nil {
		r.seen = map[string]bool{}
	}
	null := isNull(raw)
	r.seen[key] = null
	if null || r.err != nil {
		return
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		r.err = fmt.Errorf("%s: %w", key, err)
	}
}

func readRaw(r *objectReader, key string, dst *json.RawMessage) {
	raw, ok := r.raw[key]
	if !ok {
		return
	}
	delete(r.raw, key)
	if r.seen == nil {
		r.seen = map[string]bool{}
	}
	null := isNull(raw)
	r.seen[key] = null
	if null {
		return
	}
	*dst = append(json.RawMessage(nil), raw...)
}

func (r *objectReader) finish() (map[string]bool, map[string]json.RawMessage, error) {
	if r.err != nil {
		return nil, nil, r.err
	}
	if len(r.raw) == 0 {
		return r.seen, nil, nil
	}
	return r.seen, r.raw, nil
}

type objectWriter struct {
	out  map[string]json.RawMessage
	seen map[string]bool
	err  error
}

func writeObject(seen map[string]bool, extra map[string]json.RawMessage) *objectWriter {
	w := &objectWriter{out: make(map[string]json.RawMessage, len(extra)+8), seen: seen}
	for k, v := range extra {
		w.out[k] = v
	}
	return w
}

func writeField[T any](w *objectWriter, key string, v T, zero bool) {
	if w.err != nil {
		return
	}
	if zero {
		null, ok := w.seen[key]
		if !ok {
			return
		}
		if null {
			w.out[key] = json.RawMessage("null")
			return
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		w.err = fmt.Errorf("%s: %w", key, err)
		return
	}
	w.out[key] = b
}

func writeRaw(w *objectWriter, key string, v json.RawMessage, fallback string) {
	if len(v) != 0 {
		w.out[key] = v
		return
	}
	null, ok := w.seen[key]
	if !ok {
		return
	}
	if null {
		w.out[key] = json.RawMessage("null")
		return
	}
	w.out[key] = json.RawMessage(fallback)
}

func (w *objectWriter) finish() ([]byte, error) {
	if w.err != nil {
		return nil, w.err
	}
	if len(w.out) == 0 && w.seen[nullObject] {
		return []byte("null"), nil
	}
	return json.Marshal(w.out)
}
