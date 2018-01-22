// +build linux darwin freebsd netbsd

package checks

import (
	"testing"
	"time"

	"github.com/monosense-products/mackerel-agent/cmdutil"
	"github.com/monosense-products/mackerel-agent/config"
)

func TestChecker_CheckTimeout(t *testing.T) {
	checkerTimeout := Checker{
		Config: &config.CheckPlugin{
			Command: config.Command{
				Cmd: "sleep 2",
				CommandOption: cmdutil.CommandOption{
					TimeoutDuration: 1 * time.Second,
				},
			},
		},
	}

	{
		report := checkerTimeout.Check()
		if report.Status != StatusUnknown {
			t.Errorf("status should be UNKNOWN: %v", report.Status)
		}
		if report.Message != "command timed out" {
			t.Errorf("wrong message: %q", report.Message)
		}
	}
}
