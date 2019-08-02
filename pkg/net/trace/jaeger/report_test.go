package jaeger

import (
	"github.com/bilibili/kratos/pkg/net/trace"
	"net/http"
	"testing"
	"time"
)

func TestJaeger(t *testing.T) {

	c := &Config{
		HostPort:"127.0.0.1:6831",
		MaxPacketSize:10000,
	}
	report := newReport(c)

	t1 := trace.NewTracer("service1", report, true)
	t2 := trace.NewTracer("service2", report, true)
	sp1 := t1.New("option_1")
	sp1.SetTag(trace.Tag{Key:"sp1_key",Value:"sp1_val"})
	sp2 := sp1.Fork("service3", "opt_client")
	sp2.SetTag(trace.Tag{Key:"sp2_key",Value:"sp2_val"})
	// inject
	header := make(http.Header)
	t1.Inject(sp2, trace.HTTPFormat, header)
	t.Log(header)
	sp3, err := t2.Extract(trace.HTTPFormat, header)
	sp3.SetTag(trace.Tag{Key:"sp3_key",Value:"sp3_val"})
	if err != nil {
		t.Fatal(err)
	}
	sp3.Finish(nil)
	time.Sleep(time.Second*2)
	sp2.Finish(nil)
	time.Sleep(time.Second*2)
	sp1.Finish(nil)
	report.Close()
}
