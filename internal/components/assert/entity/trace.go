package entity

type TraceData struct {
	Size   int      `json:"size,omitempty"`
	Traces []*Trace `json:"traces"`
}

type Trace struct {
	TraceID string  `json:"traceId"`
	Spans   []*Span `json:"spans"`
}

type Span struct {
	TraceID       string            `json:"traceId"`
	SpanID        string            `json:"spanId"`
	Duration      IntOrString       `json:"duration"`
	Flags         IntOrString       `json:"flags"`
	OperationName string            `json:"operationName"`
	StartTime     IntOrString       `json:"startTime"`
	ParentSpanID  string            `json:"parentSpanId"`
	Logs          []any             `json:"logs"`
	Tags          map[string]string `json:"tags"`
	References    []*Reference      `json:"references"`
}

type Reference struct {
	SpanID  string `json:"spanId"`
	TraceID string `json:"traceId"`
	RefType string `json:"refType"`
}
