package debugging

// ValidateRequest validates request identity and arguments against the command schema.
func (c *Catalog) ValidateRequest(request Request) error {
	_, err := c.NormalizeRequest(request)

	return err
}
