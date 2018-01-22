package command

import (
	"github.com/monosense-products/mackerel-agent/config"
	"github.com/monosense-products/mackerel-agent/metrics"
	metricsNetbsd "github.com/monosense-products/mackerel-agent/metrics/netbsd"
	"github.com/monosense-products/mackerel-agent/spec"
	specNetbsd "github.com/monosense-products/mackerel-agent/spec/netbsd"
)

func specGenerators() []spec.Generator {
	return []spec.Generator{
		&specNetbsd.KernelGenerator{},
		&specNetbsd.MemoryGenerator{},
		&specNetbsd.CPUGenerator{},
		&spec.FilesystemGenerator{},
	}
}

func interfaceGenerator() spec.InterfaceGenerator {
	return &specNetbsd.InterfaceGenerator{}
}

func metricsGenerators(conf *config.Config) []metrics.Generator {
	generators := []metrics.Generator{
		&metrics.Loadavg5Generator{},
		&metricsNetbsd.CPUUsageGenerator{},
		&metrics.FilesystemGenerator{IgnoreRegexp: conf.Filesystems.Ignore.Regexp, UseMountpoint: conf.Filesystems.UseMountpoint},
		&metricsNetbsd.MemoryGenerator{},
		&metrics.InterfaceGenerator{Interval: metricsInterval},
	}

	return generators
}
