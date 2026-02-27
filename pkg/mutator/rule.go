package mutator

// LabelSelector matches metrics by name and/or labels.
// An empty Name matches any name. Nil/empty Labels matches any labels.
// Label matching uses AND semantics — all specified labels must match.
type LabelSelector struct {
	Name   string
	Labels map[string]string
}

// Rule pairs a LabelSelector with a Mutator to apply when the selector matches.
type Rule struct {
	Selector LabelSelector
	Mutator  Mutator
}

// Matches reports whether the given metric name and labels satisfy the selector.
func (r Rule) Matches(name string, labels map[string]string) bool {
	if r.Selector.Name != "" && r.Selector.Name != name {
		return false
	}
	for k, v := range r.Selector.Labels {
		if labels[k] != v {
			return false
		}
	}
	return true
}
