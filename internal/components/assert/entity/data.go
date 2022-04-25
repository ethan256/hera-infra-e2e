package entity

import "github.com/prometheus/prom2json"

type Data struct {
	ServiceName string              `json:"serviceName"`
	Traces      []*Trace            `json:"traces,omitempty"`
	Metrics     []*prom2json.Family `json:"metrics,omitempty"`
}
