package infrastructure

import "github.com/google/uuid"

// generator defines the interface for ID generation.
// this allows mocking in tests
type Generator interface {
	Generate() string
}

type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

func (g *UUIDGenerator) Generate() string {
	return uuid.New().String()
}
