package pgs

// GetDependents takes in a list of a Message or Enum's direct dependents and generates a
// full list of dependents including transitive dependents. Additionally, this function
// ensure that the output slice is deduped.
func GetDependents(directDeps []Message, name string) []Message {
	set := make(map[string]Message)

	for _, d := range directDeps {
		set[d.FullyQualifiedName()] = d
		for _, dd := range d.Dependents() {
			set[dd.FullyQualifiedName()] = dd
		}
	}

	// ensure that the message calling DedupeDependents is not included
	delete(set, name)

	dependents := make([]Message, 0, len(set))
	for _, d := range set {
		dependents = append(dependents, d)
	}

	return dependents
}
