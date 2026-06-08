package collector

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/stmcginnis/gofish"
)

var liteonPSURe = regexp.MustCompile(`^p(\d+)_(.+)$`)

// per-PSU: sensor Id suffix -> canonical metric <name>
var liteonPSUSensor = map[string]string{
	"vin": "input_voltage", "vout": "output_voltage",
	"iin": "input_current", "iout": "output_current",
	"pin": "input_power", "pout": "output_power",
	"ambient": "temp_input", "exhaust": "temp_output", "hotspot": "temp_hotspot",
	"fan1": "fan1",
}

// shelf-level: full sensor Id -> canonical metric <name>
var liteonShelfSensor = map[string]string{
	"chassis_input_power":    "total_power_in",
	"chassis_output_power":   "total_power_out",
	"chassis_output_current": "total_current_out",
	"ishare":                 "current_share",
	"chassis_efficiency":     "total_efficiency",
	"chassis_load":           "power_load",
	"chassis_input_current":  "total_current_in",
	"chassis_temperature":    "temp_shelf",
	"DC_temp_plus":           "dc_temp_plus",
	"DC_temp_minus":          "dc_temp_minus",
}

// TODO: SKIPPED LITE-ON metrics
//   frequency_p<N>_freqin (6, ReadingType=Frequency, ~60Hz) — these endpoints exist on the
//   device but are NOT Members of the powershelf/Sensors collection, so chassis.Sensors()
//   never returns them (and they are omitted from the trimmed test fixture). Needs either a
//   direct-by-path fetch or confirmation the live BMC lists them. Would map to per-PSU
//   input_frequency once available.

type liteonAdapter struct{}

func (a *liteonAdapter) name() string { return "liteon" }

func (a *liteonAdapter) collect(ctx context.Context, client *gofish.APIClient) ([]Sample, bool, error) {
	shelf, err := findPowershelfChassis(client, "LITE-ON")
	if err != nil {
		return nil, false, err
	}
	if shelf == nil {
		return nil, false, nil // not a Lite-On powershelf, let other adapters try
	}

	sensors, err := shelf.Sensors()
	if err != nil {
		return nil, true, err // it's a Lite-On device, but the read failed
	}

	var samples []Sample
	for _, s := range sensors {
		if s.Reading == nil {
			continue
		}
		val := *s.Reading

		if m := liteonPSURe.FindStringSubmatch(s.ID); m != nil { // per-PSU
			if name, ok := liteonPSUSensor[m[2]]; ok {
				n, _ := strconv.Atoi(m[1])
				samples = append(samples, Sample{Name: name, PowerSupply: fmt.Sprintf("ps%d", n+1), Value: val})
				continue
			}
			// unmapped per-PSU suffix → fall through to catch-all (raw id keeps the p#)
		}
		if name, ok := liteonShelfSensor[s.ID]; ok { // curated shelf
			samples = append(samples, Sample{Name: name, Value: val})
			continue
		}
		if cs, ok := catchallSample(s.ID, string(s.ReadingType), val); ok { // nothing dropped
			samples = append(samples, cs)
		}
	}

	status, err := collectPSUStatus(shelf, func(id string) string {
		n, err := strconv.Atoi(id) // "0".."5"
		if err != nil {
			return ""
		}
		return fmt.Sprintf("ps%d", n+1) // align with sensor mapping (p0 -> ps1)
	})
	if err != nil {
		return nil, true, err // it's a Lite-On device, but the PSU read failed
	}
	samples = append(samples, status...)

	return samples, true, nil
}
