package collector

type nameFilter struct {
	allow map[string]struct{}
	deny  map[string]struct{}
}

func newNameFilter(allow, deny []string) nameFilter {
	f := nameFilter{
		allow: map[string]struct{}{},
		deny:  map[string]struct{}{},
	}
	for _, a := range allow {
		f.allow[a] = struct{}{}
	}
	for _, d := range deny {
		f.deny[d] = struct{}{}
	}
	return f
}

func (f nameFilter) Allowed(name string) bool {
	if len(f.allow) > 0 {
		if _, ok := f.allow[name]; !ok {
			return false
		}
	}
	if _, denied := f.deny[name]; denied {
		return false
	}
	return true
}
