package chatter

import "crypto/tls"

// https://github.com/ssllabs/research/wiki/ssl-and-tls-deployment-best-practices
// https://gist.github.com/denji/12b3a568f092ab951456
var defaultTLSConfig = tls.Config{
	MinVersion:               tls.VersionTLS12,
	PreferServerCipherSuites: true,
	CurvePreferences: []tls.CurveID{
		tls.CurveP521,
		tls.CurveP384,
		tls.CurveP256,
	},
	CipherSuites: []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	},
}
