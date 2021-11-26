// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

package httpinstr

import (
	"encoding/json"
	"sort"
	"strings"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
)

// setEventSpanTags sets the security event span tags into the service entry span.
func setEventSpanTags(span ddtrace.Span, events json.RawMessage) {
	// Set the appsec event span tag
	// eventTag is the structure to use in the `_dd.appsec.json` span tag.
	type eventTag struct {
		Triggers json.RawMessage `json:"triggers"`
	}
	// TODO(Julio-Guerra): avoid serializing the json in the request hot path
	event, err := json.Marshal(eventTag{Triggers: events})
	if err != nil {
		log.Error("appsec: unexpected error while serializing the appsec event span tag: %v", err)
		return
	}
	span.SetTag("_dd.appsec.json", string(event))
	// Keep this span due to the security event
	span.SetTag(ext.ManualKeep, true)
	span.SetTag("_dd.origin", "appsec")
	// Set the appsec.event tag needed by the appsec backend
	span.SetTag("appsec.event", true)
}

// setEventSpanTags sets the AppSec-specific span tags when a security event occurred into the service entry span.
func setSpanTags(span ddtrace.Span, events json.RawMessage, remoteIP string, headers map[string][]string) {
	setEventSpanTags(span, events)
	span.SetTag("network.client.ip", remoteIP)
	for h, v := range normalizeHTTPHeaders(headers) {
		span.SetTag("http.request.headers."+h, v)
	}
}

// List of HTTP headers we collect and send.
var collectedHTTPHeaders = [...]string{
	"host",
	"x-forwarded-for",
	"x-client-ip",
	"x-real-ip",
	"x-forwarded",
	"x-cluster-client-ip",
	"forwarded-for",
	"forwarded",
	"via",
	"true-client-ip",
	"content-length",
	"content-type",
	"content-encoding",
	"content-language",
	"forwarded",
	"user-agent",
	"accept",
	"accept-encoding",
	"accept-language",
}

func init() {
	// Required by sort.SearchStrings
	sort.Strings(collectedHTTPHeaders[:])
}

// normalizeHTTPHeaders returns the HTTP headers following Datadog's normalization format.
func normalizeHTTPHeaders(headers map[string][]string) (normalized map[string]string) {
	if len(headers) == 0 {
		return nil
	}
	normalized = make(map[string]string)
	for k, v := range headers {
		if i := sort.SearchStrings(collectedHTTPHeaders[:], k); i < len(collectedHTTPHeaders) && collectedHTTPHeaders[i] == k {
			normalized[k] = strings.Join(v, ",")
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}
