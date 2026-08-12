package collector

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	CommonStateHelp           = "1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)"
	CommonHealthHelp          = "1(OK),2(Warning),3(Critical)"
	CommonSeverityHelp        = CommonHealthHelp
	CommonLinkHelp            = "1(LinkUp),2(NoLink),3(LinkDown)"
	CommonNVLinkPortLinkHelp  = "1(LinkUp),2(Starting),3(Training),4(LinkDown),5(NoLink)"
	CommonPortLinkHelp        = "1(Up),0(Down)"
	CommonIntrusionSensorHelp = "1(Normal),2(TamperingDetected),3(HardwareIntrusion)"
	// CommonDetectorStateHelp describes leak detector states. NOTE: unlike the health
	// encoding, values above 3 do NOT mean "worse than critical" - 4/5 indicate the
	// detector is not reporting, and only exist from LeakDetector v1_6_0 (see
	// parseDetectorState). Alert on == 3 for a leak, never >= 2.
	CommonDetectorStateHelp = "1(OK),2(Warning),3(Critical),4(Unavailable),5(Absent)"
)

type Metric struct {
	desc *prometheus.Desc
}

func addToMetricMap(metricMap map[string]Metric, subsystem, name, help string, variableLabels []string) {
	metricKey := fmt.Sprintf("%s_%s", subsystem, name)
	metricMap[metricKey] = Metric{
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, name),
			help,
			variableLabels,
			nil,
		),
	}
}
