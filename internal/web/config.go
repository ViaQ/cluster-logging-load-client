package web

type ServerConfig struct {
	ListenAddress string     `yaml:"listenAddress"`
	TLS           *TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
	CertificateFile    string `yaml:"certificateFile"`
	KeyFile            string `yaml:"keyFile"`
}
