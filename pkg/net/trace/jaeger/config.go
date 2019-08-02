package jaeger

import (
	"github.com/bilibili/kratos/pkg/conf/env"
	"github.com/bilibili/kratos/pkg/net/trace"
)

type Config struct {
	HostPort string
	MaxPacketSize int
	DisableSample bool
}


func Init(c *Config){
	if(c.MaxPacketSize == 0){
		c.MaxPacketSize = 10000
	}
	if(c.HostPort == ""){
		c.HostPort = "127.0.0.1:6831"
	}
	trace.SetGlobalTracer(trace.NewTracer(env.AppID, newReport(c), c.DisableSample))
}