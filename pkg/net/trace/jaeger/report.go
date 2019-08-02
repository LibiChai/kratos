package jaeger

import (
	"fmt"
	"github.com/bilibili/kratos/pkg/net/trace"
	j "github.com/bilibili/kratos/pkg/net/trace/jaeger/thrift-gen/jaeger"
	protogen "github.com/bilibili/kratos/pkg/net/trace/proto"
	"github.com/pkg/errors"
	"time"
)

type report struct {
	transport *udpSender
}

func newReport(c *Config) *report{
	t,err  := NewUDPTransport(c.HostPort,c.MaxPacketSize)
	if(err != nil){
		panic(errors.Wrap(err,"jeager report init faile"))
	}
	return &report{t}
}


func (r *report)WriteSpan(raw *trace.Span) (err error) {

	_,err = r.transport.Append(r.buildJaegerSpan(raw))
	if(err != nil){
		return err
	}
	_,err = r.transport.Flush()
	if(err != nil){
		return err
	}

	return nil
}

func (r *report) Close() error {
	return r.transport.Close()
}

func (r *report)buildJaegerSpan(raw *trace.Span)(process *j.Process,span *j.Span){
	fmt.Printf("%#v \n",raw.ServiceName())
	startTime := raw.StartTime().UnixNano() / int64(time.Microsecond)
	duration := raw.Duration() / time.Microsecond

	span = &j.Span{
		TraceIdLow:    int64(raw.Context().TraceID),
		TraceIdHigh:   int64(raw.Context().TraceID),
		SpanId:        int64(raw.Context().SpanID),
		ParentSpanId:  int64(raw.Context().ParentID),
		OperationName: raw.OperationName(),
		Flags:         int32(raw.Context().Flags),
		StartTime:     startTime,
		Duration:      int64(duration),
		Tags:          buildTags(raw.Tags()),
		Logs:          buildLogs(raw.Logs()),
	}

	process = &j.Process{
		ServiceName: raw.ServiceName(),
	}

	return
}


func buildTags(tags []trace.Tag) []*j.Tag {
	jTags := make([]*j.Tag, 0, len(tags))
	for _, tag := range tags {
		jTag := buildTag(&tag)
		jTags = append(jTags, jTag)
	}
	return jTags
}

func buildLogs(logs []*protogen.Log) []*j.Log {
	jLogs := make([]*j.Log, 0, len(logs))

	for _, log := range logs {
		jLog := &j.Log{
			Timestamp: log.Timestamp / int64(time.Microsecond),
			Fields:    convertLogsToJaegerTags(log.Fields),
		}
		jLogs = append(jLogs, jLog)
	}
	return jLogs
}

func buildTag(tag *trace.Tag ) *j.Tag {
	jTag := &j.Tag{Key: tag.Key}
	switch value := tag.Value.(type) {
	case string:
		vStr := value
		jTag.VStr = &vStr
		jTag.VType = j.TagType_STRING
	case []byte:
		jTag.VBinary = value
		jTag.VType = j.TagType_BINARY
	case int:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case uint:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case int8:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case uint8:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case int16:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case uint16:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case int32:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case uint32:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case int64:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case uint64:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case float32:
		vDouble := float64(value)
		jTag.VDouble = &vDouble
		jTag.VType = j.TagType_DOUBLE
	case float64:
		vDouble := float64(value)
		jTag.VDouble = &vDouble
		jTag.VType = j.TagType_DOUBLE
	case bool:
		vBool := value
		jTag.VBool = &vBool
		jTag.VType = j.TagType_BOOL
	default:
		vStr :=  fmt.Sprint(value)
		jTag.VStr = &vStr
		jTag.VType = j.TagType_STRING
	}
	return jTag
}
func convertLogsToJaegerTags(logFields []*protogen.Field) []*j.Tag {
	fields := make([]*j.Tag,len(logFields))
	for i, logField := range logFields {
		tag := &trace.Tag{
			Key:logField.Key,
			Value:string(logField.Value),
		}
		fields[i] = buildTag(tag)
	}
	return fields
}
