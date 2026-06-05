package collector

import (
	"context"
	"regexp"

	"github.com/stmcginnis/gofish"
)

var deltaPSURe = regexp.MustCompile(`^ps(\d+)_(.+)$`)

type deltaAdapter struct{}

func (a *deltaAdapter) name() string { return "delta" }

func (a *deltaAdapter) collect(ctx context.Context, client *gofish.APIClient) ([]Sample, bool, error) {
	shelf, err := findPowershelfChassis(client, "DELTA")
	if err != nil {
		return nil, false, err
	}
	if shelf == nil {
		return nil, false, nil
	}

	sensors, err := shelf.Sensors()
	if err != nil {
		return nil, true, err
	}
	var samples []Sample
	for _, s := range sensors {
		if s.Reading == nil {
			continue
		}
		if m := deltaPSURe.FindStringSubmatch(s.ID); m != nil { // "ps1_input_current"
			samples = append(samples, Sample{Name: m[2], PowerSupply: "ps" + m[1], Value: *s.Reading})
			continue
		}
		samples = append(samples, Sample{Name: s.ID, Value: *s.Reading}) // shelf id IS the canonical name
	}

	status, err := collectPSUStatus(shelf, func(id string) string {
		return "ps" + id // Delta PSU Ids are already "1".."6"
	})
	if err != nil {
		return nil, true, err
	}
	samples = append(samples, status...)

	return samples, true, nil
}
