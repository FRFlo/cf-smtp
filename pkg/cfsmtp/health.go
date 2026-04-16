package cfsmtp

type HealthStatus struct {
	Status       string `json:"status"`
	Service      string `json:"service"`
	ListenAddr   string `json:"listen_addr"`
	Hostname     string `json:"hostname"`
	MaxMessageMB int64  `json:"max_message_mb"`
}

func Healthcheck(config Config) HealthStatus {
	return HealthStatus{
		Status:       "ok",
		Service:      "cf-smtp",
		ListenAddr:   config.ListenAddr,
		Hostname:     config.Hostname,
		MaxMessageMB: config.MaxMessageBytes / (1024 * 1024),
	}
}
