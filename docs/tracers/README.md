Tracer Types
============

This document was generated with `benthos --list-tracers`

A tracer type represents a destination for Benthos to send opentracing events to
such as Jaeger.

Many Benthos components create spans on messages passing through a pipeline, and
so opentracing is a great way to analyse the pathways of individual messages as
they progress through a Benthos instance.

WARNING: Although the configuration spec of this component is stable the format
of spans, tags and logs created by Benthos is subject to change as it is tuned
for improvement.

## `jaeger`

``` yaml
type: jaeger
jaeger:
  agent_address: localhost:6831
  flush_interval: ""
  service_name: benthos
  span_sample: 1
```

Send spans to a Jaeger agent.

A static span sample can be set anywhere between 0 and 1.

## `none`

``` yaml
type: none
none: {}
```

Do not send opentracing events anywhere.
