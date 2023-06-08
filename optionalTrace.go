package main

import (
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func optionalTrace() {

	ctx, err := initTracer()
	if err != nil {
		log.Panic(err)
	}
	tracer := tp.Tracer("example/otel-go-batch")
	// defer func() { _ = tp.Shutdown(ctx) }()

	// make a little tree of spans
	var span trace.Span
	ctx, span = tracer.Start(ctx, "Evaluate the queue and environment")
	defer span.End()

	startupTraceSpanLink := trace.Link{
		SpanContext: span.SpanContext(),
		Attributes: []attribute.KeyValue{
			attribute.String("name", "Link to job start"),
		},
	}

	jobs := make([]int, 1000)

	errorsFound := 0
	errorTypesStr := []string{"SchedulingFailure", "RecoverableStartupFailure", "ConnectionFailure", "TooManyPuppies"}

	for i := range jobs {
		// do the jobs and collect up the errors
		errIntArray := doSomeDetailedJobWork2(int64(i))
		if errIntArray != nil {
			var tracerWorker = tp.Tracer("example/otel-go-batch")
			_, spanWorker := tracerWorker.Start(ctx, "Errors Summary Span", trace.WithLinks(startupTraceSpanLink))
			defer spanWorker.End()
			spanWorker.SetAttributes(attribute.String("job.emitted_by", "scheduler"))
			spanWorker.SetAttributes(attribute.Int64("job.number", int64(i)))
			spanWorker.SetStatus(codes.Error, "Error summary span")
			// make a span for each error
			for errorKind, errorCount := range errIntArray {
				errorAttrName := fmt.Sprintf("error.%s", errorTypesStr[errorKind])
				spanWorker.SetAttributes(attribute.Int64(errorAttrName, int64(errorCount)))
			}

			spanWorker.End()
		}
	}

	// do all the error reporting at the end.

	fmt.Printf("jobs run %d, errors found %d \n", len(jobs), errorsFound)
	span.End()
	defer func() { _ = tp.Shutdown(ctx) }()
	time.Sleep(2 * time.Second)
}
