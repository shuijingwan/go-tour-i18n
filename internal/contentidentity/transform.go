package contentidentity

import "bytes"

// MemoizationActive reports only the lifetime of the explicit transformation
// scope, so callers can retain their canonical eager path outside an invocation.
func MemoizationActive() bool { return activeMemo() != nil }

// Transform reuses a pure byte transformation in the current memoization
// scope. The caller supplies a versioned operation name and its complete input
// bytes (including all options). It must not read mutable state in compute.
// Fresh filesystem observations, expected hashes and current eligibility gates
// belong outside this function and must still run on every call.
//
// Complete bytes are compared, not just an index/hash. Cached inputs and outputs
// never alias caller buffers. Errors and oversized results are never cached;
// without an invocation scope this simply calls compute. Within a scope an
// implementation/content-addressed disposable L2 may supply the same pure byte
// result. Missing, invalid or unavailable storage always falls back to compute.
func Transform(operation string, input []byte, compute func() ([]byte, error)) ([]byte, error) {
	m := activeMemo()
	if m == nil || operation == "" {
		return compute()
	}
	m.Lock()
	if e := m.entry(input, false); e != nil {
		if output, ok := e.transforms[operation]; ok {
			out := bytes.Clone(output)
			m.Unlock()
			return out, nil
		}
	}
	m.Unlock()
	var output []byte
	var err error
	key := ""
	found := false
	if m.store != nil {
		key = m.store.key(operation, input)
		output, found = m.store.load(key)
	}
	if !found {
		output, err = compute()
	}
	if err != nil {
		return output, err
	}
	if !found && key != "" {
		m.store.save(key, output)
	}
	m.Lock()
	defer m.Unlock()
	cost := len(output) + len(operation) + 64
	if m.transformBytes+cost <= 16<<20 {
		e := m.entry(input, true)
		if e == nil {
			e = m.transformInput(input, cost)
		}
		if e != nil {
			if _, exists := e.transforms[operation]; !exists {
				if e.transforms == nil {
					e.transforms = map[string][]byte{}
				}
				e.transforms[operation] = bytes.Clone(output)
				m.transformBytes += cost
			}
		}
	}
	return output, nil
}
