// Package telemetrics sets context to telemetry-related value(s) from a given request's context.
//
// The following services, and their related headers, are considered to be telemetry-related:
//
//   - Istio
//   - Jaeger
//   - Envoy Proxy
//   - Zipkin
//   - Otel
//   - AWS X-Ray
//
// The package additionally provides middleware for adding request-specific route context.
package telemetrics
