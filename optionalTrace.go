package main

import (
	"context"
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

	startupTraceSpanLink := trace.Link{
		SpanContext: span.SpanContext(),
		Attributes: []attribute.KeyValue{
			attribute.String("name", "Link to job start"),
		},
	}
	span.End()

	jobs := make([]int, 10)
	var errStringArray []string
	errorsFound := 0

	for i := range jobs {
		// do the jobs and collect up the errors
		errStrings := doSomeDetailedJobWork(int64(i))
		if errStrings != nil {
			errStringArray = append(errStringArray, errStrings...)
		}
	}

	// do all the error reporting at the end.
	if len(errStringArray) > 0 {
		errorsFound += len(errStringArray)
		var tracerWorker = tp.Tracer("example/otel-go-batch")
		ctxWorker, spanWorker := tracerWorker.Start(context.Background(), "Something went wrong", trace.WithLinks(startupTraceSpanLink))
		defer spanWorker.End()
		spanWorker.SetAttributes(attribute.String("job.emitted_by", "scheduler"))

		// make a span for each error
		for _, errStr := range errStringArray {
			_, span := tracerWorker.Start(ctxWorker, "error in job")
			span.SetAttributes(attribute.String("job.emitted_by", "scheduler"))
			// if you want to capture the errors without setting the status, use the commented line
			// span.SetAttributes(attributes.String("error.message", errStr)
			span.SetStatus(codes.Error, errStr)
			span.End()
			time.Sleep(time.Millisecond)
		}

		spanWorker.End()
	}
	fmt.Printf("jobs run %d, errors found %d \n", len(jobs), errorsFound)
	defer func() { _ = tp.Shutdown(ctx) }()
	time.Sleep(2 * time.Second)
}
