package scenarios

import "encoding/json"

// Bind validates launch values, applies defaults, and produces concrete ordered steps.
func (definition Definition) Bind(values map[string]json.RawMessage) ([]BoundStep, error) {
	steps, _, err := definition.BindWithValues(values)

	return steps, err
}

// Load reads and validates one source-qualified scenario definition.
func (repository *Repository) Load(qualifiedReference string) (Definition, error) {
	record, err := repository.LoadRecord(qualifiedReference)
	if err != nil {
		return Definition{}, err
	}

	return record.Definition, nil
}

// Duplicate copies a current source definition to an absent destination.
func (repository *Repository) Duplicate(sourceReference, sourceRevision, destinationReference string) (Record, error) {
	return repository.DuplicateAs(sourceReference, sourceRevision, destinationReference, "")
}
