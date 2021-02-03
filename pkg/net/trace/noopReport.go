package trace

type noopReport struct {
}

func (np *noopReport) WriteSpan(sp *Span) error {
	return nil
}
func (np *noopReport) Close() error {
	return nil
}
