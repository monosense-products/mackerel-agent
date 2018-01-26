package command

import (
	"github.com/monosense-products/mackerel-agent/config"
	"github.com/monosense-products/mackerel-agent/metrics"
	metricsLinux "github.com/monosense-products/mackerel-agent/metrics/linux"
	"github.com/monosense-products/mackerel-agent/spec"
	specLinux "github.com/monosense-products/mackerel-agent/spec/linux"
)

func specGenerators() []spec.Generator {
	return []spec.Generator{
		&specLinux.KernelGenerator{},
		&specLinux.CPUGenerator{},
		&specLinux.MemoryGenerator{},
		&specLinux.BlockDeviceGenerator{},
		&spec.FilesystemGenerator{},
	}
}

func interfaceGenerator() spec.InterfaceGenerator {
	return &specLinux.InterfaceGenerator{}
}

func metricsGenerators(conf *config.Config) []metrics.Generator {
	generators := []metrics.Generator{
		&metrics.Loadavg5Generator{},
		&metricsLinux.CPUUsageGenerator{Interval: metricsInterval},
		&metricsLinux.MemoryGenerator{},
		&metrics.InterfaceGenerator{Interval: metricsInterval},
		&metricsLinux.DiskGenerator{Interval: metricsInterval, UseMountpoint: conf.Filesystems.UseMountpoint},
		&metrics.FilesystemGenerator{IgnoreRegexp: conf.Filesystems.Ignore.Regexp, UseMountpoint: conf.Filesystems.UseMountpoint},
	}

	return generators
}
