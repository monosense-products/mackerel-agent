package agent

import (
	"sync"

	"github.com/mackerelio/golib/logging"
	"github.com/monosense-products/mackerel-agent/metrics"
)

var logger = logging.GetLogger("agent")

func generateValues(generators []metrics.Generator) []*metrics.ValuesCustomIdentifier {
	processed := make(chan *metrics.ValuesCustomIdentifier)
	finish := make(chan struct{})
	result := make(chan []*metrics.ValuesCustomIdentifier)

	go func() {
		allValues := []*metrics.ValuesCustomIdentifier{}
		for {
			select {
			case values := <-processed:
				allValues = metrics.MergeValuesCustomIdentifiers(allValues, values)
			case <-finish:
				result <- allValues
				return
			}
		}
	}()

	go func() {
		var wg sync.WaitGroup
		for _, g := range generators {
			wg.Add(1)
			go func(g metrics.Generator) {
				defer func() {
					if r := recover(); r != nil {
						logger.Errorf("Panic: generating value in %T (skip this metric): %s", g, r)
					}
					wg.Done()
				}()

				values, err := g.Generate()
				if err != nil {
					logger.Errorf("Failed to generate value in %T (skip this metric): %s", g, err.Error())
					return
				}
				var customIdentifier *string
				if pluginGenerator, ok := g.(metrics.PluginGenerator); ok {
					customIdentifier = pluginGenerator.CustomIdentifier()
				}
				processed <- &metrics.ValuesCustomIdentifier{
					Values:           values,
					CustomIdentifier: customIdentifier,
				}
			}(g)
		}
		wg.Wait()
		finish <- struct{}{} // processed all jobs
	}()

	return <-result
}
