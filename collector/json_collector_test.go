package collector

import (
	"context"
	"testing"
	"time"

	"github.com/itchyny/gojq"
	gta "gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func Test_metricsFromBody(t *testing.T) {
	tT := map[string]struct {
		timeout       time.Duration
		rawBody       []byte
		jqFilter      string
		wantErrString string
		wantMetrics   []JSONYieldedMetric
	}{
		"happy path for deltaenergysystems Oem": {
			timeout: 5 * time.Second,
			rawBody: []byte(`{
"Oem": {
  "deltaenergysystems": {
    "AllSensors": {
      "Sensors": [{
        "DataSourceUri": "/redfish/v1/Chassis/PowerShelf_0/Sensors/hotswap_temp",
        "DeviceName": "hotswap_temp",
        "Reading": 43.0,
        "ReadingType": "Temperature",
        "ReadingUnits": "Cel"
      }]
    }
  }
}}`),
			jqFilter: `[.Oem.deltaenergysystems.AllSensors.Sensors[]] | map({
        name: (if .DeviceName | test("^ps[0-9]+_") then .DeviceName | sub("^ps[0-9]+_"; "") else .DeviceName end),
        value: .Reading,
        labels: (
          if .DeviceName | test("^ps[0-9]+_") then {"power_supply_id": (.DeviceName | split("_")[0])}
          else {}
          end),
          _raw: .
      }) |
      map(
        .help = "Value yielded from the Redfish API endpoint: " + ._raw.DataSourceUri + ",type: " + ._raw.ReadingType + ",unit: " + ._raw.ReadingUnits) |
        map(del(._raw)) | sort_by(.name)`,
			wantErrString: "",
			wantMetrics: []JSONYieldedMetric{
				{
					Name:   "hotswap_temp",
					Value:  43.0,
					Help:   "Value yielded from the Redfish API endpoint: /redfish/v1/Chassis/PowerShelf_0/Sensors/hotswap_temp,type: Temperature,unit: Cel",
					Labels: map[string]string{},
				},
			},
		},
		"errors are bubbled up": {
			timeout: 5 * time.Second,
			rawBody: []byte(`{
"Oem": {
  "deltaenergysystems": {
    "AllSensors": {
      "Sensors": [{
        "DataSourceUri": "/redfish/v1/Chassis/PowerShelf_0/Sensors/hotswap_temp",
        "DeviceName": "hotswap_temp",
        "Reading": 43.0,
        "ReadingType": "Temperature",
        "ReadingUnits": "Cel"
      }]
    }
  }
}}`),
			jqFilter: `[.Oem.deltaenergysystems.AllSensors.Sensors[]] | map({
        name1: (if .DeviceName | test("^ps[0-9]+_") then .DeviceName | sub("^ps[0-9]+_"; "") else .DeviceName end),
        valuefoo: .Reading,
        labels: (
          if .DeviceName | test("^ps[0-9]+_") then {"power_supply_id": (.DeviceName | split("_")[0])}
          else {}
          end),
          _raw: .
      }) |
      map(
        .help = "Value yielded from the Redfish API endpoint: " + ._raw.DataSourceUri + ",type: " + ._raw.ReadingType + ",unit: " + ._raw.ReadingUnits) |
        map(del(._raw)) | sort_by(.name)`,
			wantErrString: "item missing name, provided keys: [help labels name1 valuefoo]",
			wantMetrics:   nil,
		},
		"context timeout is an error": {
			timeout:       0,
			rawBody:       []byte(`{}`),
			jqFilter:      `.`,
			wantErrString: "context deadline exceeded",
			wantMetrics:   []JSONYieldedMetric{},
		},
	}
	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			query, err := gojq.Parse(test.jqFilter)
			if err != nil {
				gta.Assert(t, cmp.ErrorContains(err, test.wantErrString))
			}
			ctx, cancel := context.WithTimeout(t.Context(), test.timeout)
			defer cancel()

			got, err := metricsFromBody(ctx, query, test.rawBody)
			if err != nil {
				gta.Assert(t, cmp.ErrorContains(err, test.wantErrString))
			}
			gta.Assert(t, cmp.DeepEqual(test.wantMetrics, got))
		})
	}

}

func Test_convertToMetric(t *testing.T) {
	tT := map[string]struct {
		item       map[string]any
		wantMetric JSONYieldedMetric
		wantError  string
	}{
		"normal, no labels": {
			item: map[string]any{
				"name":  "foo",
				"value": 1.0,
				"help":  "bar",
			},
			wantMetric: JSONYieldedMetric{
				Name:   "foo",
				Help:   "bar",
				Value:  1.0,
				Labels: map[string]string{},
			},
			wantError: "",
		},
		"normal, labels and help": {
			item: map[string]any{
				"name":  "foo",
				"help":  "bar",
				"value": 1.0,
				"labels": map[string]any{
					"tree": "house",
				},
			},
			wantMetric: JSONYieldedMetric{
				Name:  "foo",
				Help:  "bar",
				Value: 1.0,
				Labels: map[string]string{
					"tree": "house",
				},
			},
			wantError: "",
		},
		"unexpected input leads to empty metric and error": {
			item: map[string]any{
				"name":  1.0,
				"value": "foo",
			},
			wantMetric: JSONYieldedMetric{
				Labels: map[string]string{},
			},
			wantError: "item contained a non-string name",
		},
		"missing input leads to empty metric and error": {
			item: map[string]any{
				"foo": "name",
			},
			wantMetric: JSONYieldedMetric{
				Labels: map[string]string{},
			},
			wantError: "item missing name, provided keys: [foo]\nitem missing value, provided keys: [foo]\nitem missing help, provided keys: [foo]",
		},
	}

	for tName, test := range tT {
		t.Run(tName, func(t *testing.T) {
			got, err := convertToMetric(test.item)
			gta.Assert(t, cmp.DeepEqual(test.wantMetric, got))
			if test.wantError != "" {
				gta.Assert(t, cmp.ErrorContains(err, test.wantError))
			} else {
				gta.NilError(t, err)
			}
		})
	}
}
