package main

import (
	"context"
	"fmt"
	"log"

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

	jobs := []int{1, 2, 3, 4, 5, 6, 7}

	for i := range jobs {
		errStringArray := doSomeDetailedJobWork(int64(i))
		if err != nil {
			var tracerWorker = tp.Tracer("example/otel-go-batch")
			ctxWorker, spanWorker := tracerWorker.Start(context.Background(), "Something went wrong", trace.WithLinks(startupTraceSpanLink))
			defer spanWorker.End()
			spanWorker.SetAttributes(attribute.Int("job.number", i))
			spanWorker.SetAttributes(attribute.String("job.emitted_by", "scheduler"))

			// make a span for each error
			for _, errStr := range errStringArray {
				_, span := tracerWorker.Start(ctxWorker, "error in job")

				// if you want to capture the errors without setting the status, use the commented line
				// span.SetAttributes(attributes.String("error.message", errStr)
				span.SetStatus(codes.Error, errStr)
			}

			spanWorker.End()
			spanWorker.SetStatus(codes.Error, fmt.Sprintf("An error during lengthy job %v", i))
		}

	}
	defer func() { _ = tp.Shutdown(ctx) }()
}

// nowhere in here are we emitting spans.
// the return string array is collecting a list of errors.
func doSomeDetailedJobWork(jobNumber int64) []string {
	// the majority have no errors
	if seededRand.Intn(100) < 50 {
		return nil
	}

	if jobNumber%17 == 0 {
		return []string{"couldn't start"}
	}
	log.Printf("starting job %d \n", jobNumber)

	var errorRecord []string

	// randomly return error statuses
	if seededRand.Intn(100) < 12 {
		errorRecord = append(errorRecord, "problems at the beginning")
	}

	// loop through abunch of tasks
	loops := seededRand.Intn(1000) + 2
	for i := 0; i < loops; i += 1 {
		if seededRand.Intn(100) < 12 {
			errorRecord = append(errorRecord, fmt.Sprintf("error in job %v on task %v", jobNumber, i))
		}
	}

	return errorRecord
}
