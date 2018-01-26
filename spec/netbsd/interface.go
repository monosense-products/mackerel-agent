// +build netbsd

package netbsd

import "github.com/monosense-products/mackerel-agent/spec"

// InterfaceGenerator XXX
type InterfaceGenerator struct {
}

// Key XXX
func (g *InterfaceGenerator) Key() string {
	return "interface"
}

// Generate XXX
func (g *InterfaceGenerator) Generate() ([]spec.NetInterface, error) {
	// TODO
	return []spec.NetInterface{}, nil
}
