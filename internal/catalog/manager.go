package catalog

// Manager returns the catalog for the manager collector.
//
// See internal/collector/manager_collector.go for the emission code and
// gofish traversal that backs these entries.
func Manager() Module {
	return Module{
		Name: "manager",
		Entries: []Entry{
			{
				Name:       "redfish_manager_state",
				Help:       "manager state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
				MetricType: MetricGauge,
				ValueType:  ValueIntEnum,
				Enum:       EnumCommonState,
				Value: Source{
					Path:  "/redfish/v1/Managers/{manager_id}",
					Field: "Status.State",
				},
				Labels: managerLabels,
			},
			{
				Name:       "redfish_manager_health_state",
				Help:       "manager health,1(OK),2(Warning),3(Critical)",
				MetricType: MetricGauge,
				ValueType:  ValueIntEnum,
				Enum:       EnumCommonHealth,
				Value: Source{
					Path:  "/redfish/v1/Managers/{manager_id}",
					Field: "Status.Health",
				},
				Labels: managerLabels,
			},
			{
				Name:       "redfish_manager_power_state",
				Help:       "manager power state",
				MetricType: MetricGauge,
				ValueType:  ValueIntEnum,
				Enum:       EnumPowerState,
				Value: Source{
					Path:  "/redfish/v1/Managers/{manager_id}",
					Field: "PowerState",
				},
				Labels: managerLabels,
			},
			{
				Name:       "redfish_manager_log_service_state",
				Help:       "manager log service state,1(Enabled),2(Disabled),3(StandbyOffinline),4(StandbySpare),5(InTest),6(Starting),7(Absent),8(UnavailableOffline),9(Deferring),10(Quiesced),11(Updating),12(Standby)",
				MetricType: MetricGauge,
				ValueType:  ValueIntEnum,
				Enum:       EnumCommonState,
				Value: Source{
					Path:  "/redfish/v1/Managers/{manager_id}/LogServices/{log_service_id}",
					Field: "Status.State",
				},
				Labels: managerLogServiceLabels,
			},
			{
				Name:       "redfish_manager_log_service_health_state",
				Help:       "manager log service health state,1(OK),2(Warning),3(Critical)",
				MetricType: MetricGauge,
				ValueType:  ValueIntEnum,
				Enum:       EnumCommonHealth,
				Value: Source{
					Path:  "/redfish/v1/Managers/{manager_id}/LogServices/{log_service_id}",
					Field: "Status.Health",
				},
				Labels: managerLogServiceLabels,
			},
		},
	}
}

var managerLabels = []Label{
	{
		Name:        "manager_id",
		Field:       "Id",
		Description: "Redfish manager identifier; matches the {manager_id} URL segment.",
	},
	{
		Name:        "name",
		Field:       "Name",
		Description: "Human-readable manager name (e.g. \"BMC\").",
	},
	{
		Name:        "model",
		Field:       "Model",
		Description: "Manager hardware model string as reported by the BMC vendor.",
	},
	{
		Name:        "type",
		Field:       "ManagerType",
		Description: "Redfish ManagerType enumeration (e.g. \"BMC\", \"EnclosureManager\").",
	},
	{
		Name:        "firmware_version",
		Field:       "FirmwareVersion",
		Description: "Firmware version string of the manager.",
	},
}

var managerLogServiceLabels = []Label{
	{
		Name:        "manager_id",
		Path:        "/redfish/v1/Managers/{manager_id}",
		Field:       "Id",
		Description: "Redfish manager identifier this log service belongs to.",
	},
	{
		Name:        "log_service",
		Field:       "Name",
		Description: "Human-readable log service name.",
	},
	{
		Name:        "log_service_id",
		Field:       "Id",
		Description: "Redfish log service identifier; matches the {log_service_id} URL segment.",
	},
	{
		Name:        "log_service_enabled",
		Field:       "ServiceEnabled",
		Description: "Whether the log service is enabled (\"true\"/\"false\" as string).",
	},
	{
		Name:        "log_service_overwrite_policy",
		Field:       "OverWritePolicy",
		Description: "Log ring-buffer overwrite policy (e.g. \"WrapsWhenFull\", \"NeverOverwrites\").",
	},
}
