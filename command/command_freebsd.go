package command

import (
	"github.com/monosense-products/mackerel-agent/config"
	"github.com/monosense-products/mackerel-agent/metrics"
	metricsFreebsd "github.com/monosense-products/mackerel-agent/metrics/freebsd"
	"github.com/monosense-products/mackerel-agent/spec"
	specFreebsd "github.com/monosense-products/mackerel-agent/spec/freebsd"
)

func specGenerators() []spec.Generator {
	return []spec.Generator{
		&specFreebsd.KernelGenerator{},
		&specFreebsd.MemoryGenerator{},
		&specFreebsd.CPUGenerator{},
		&spec.FilesystemGenerator{},
	}
}

func interfaceGenerator() spec.InterfaceGenerator {
	return &specFreebsd.InterfaceGenerator{}
}

func metricsGenerators(conf *config.Config) []metrics.Generator {
	generators := []metrics.Generator{
		&metrics.Loadavg5Generator{},
		&metricsFreebsd.CPUUsageGenerator{},
		&metrics.FilesystemGenerator{IgnoreRegexp: conf.Filesystems.Ignore.Regexp, UseMountpoint: conf.Filesystems.UseMountpoint},
		&metricsFreebsd.MemoryGenerator{},
		&metrics.InterfaceGenerator{Interval: metricsInterval},
	}

	return generators
}
