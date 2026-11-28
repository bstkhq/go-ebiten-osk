package osk

// Kind describes what a field takes.
type Kind uint8

const (
	KindText Kind = iota
	KindEmail
	KindNumber
)
