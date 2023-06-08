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

	jobs := make([]int, 1000)
	errorsFound := 0

	for i := range jobs {
		// save the timestamp of starting the job just in case we need it
		startTime := time.Now()
		errStrings := doSomeDetailedJobWork(int64(i))
		if errStrings != nil {
			// if there are errors, let's do something about it
			var tracerWorker = tp.Tracer("example/otel-go-batch")
			_, spanWorker := tracerWorker.Start(ctx, "Something went wrong", trace.WithTimestamp(startTime))
			spanWorker.SetAttributes(attribute.String("job.emitted_by", "scheduler"))
			spanWorker.SetAttributes(attribute.Int64("errors.count", int64(len(errStrings))))
			// for _, errStr := range errStrings {
			// Could perform more analysis on the strings if something can be summarized.
			// }
			spanWorker.SetStatus(codes.Error, "Error Summary")
			spanWorker.End()
			time.Sleep(time.Millisecond)
		}
	}
	fmt.Printf("jobs run %d, errors found %d \n", len(jobs), errorsFound)
	span.End()
	defer func() { _ = tp.Shutdown(ctx) }()
	time.Sleep(2 * time.Second)
}
