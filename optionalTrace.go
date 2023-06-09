package main

import (
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

	successes := 0
	success_duration := 0
	failures := 0

	jobs := make([]int, 1000)

	for i := range jobs {
		// save the timestamp of starting the job just in case we need it
		startTime := time.Now()
		errStrings := doSomeDetailedJobWork(int64(i))
		if errStrings != nil {
			// if there are errors, let's do something about it
			failures += 1
			_, spanWorker := tracer.Start(ctx, "Something went wrong", trace.WithTimestamp(startTime))
			spanWorker.SetAttributes(attribute.String("job.emitted_by", "scheduler"))
			spanWorker.SetAttributes(attribute.Int64("job.number", int64(i)))
			spanWorker.SetAttributes(attribute.Int64("errors.count", int64(len(errStrings))))
			// for _, errStr := range errStrings {
			// Could perform more analysis on the strings if something can be summarized.
			// }
			spanWorker.SetStatus(codes.Error, "Error Summary")
			spanWorker.End(trace.WithTimestamp(time.Now()))
			time.Sleep(50 * time.Millisecond)
		} else {
			successes += 1
			success_duration += int(time.Since(startTime).Milliseconds())
		}
	}
	span.SetAttributes(attribute.Int64("jobs.success", int64(successes)))
	span.SetAttributes(attribute.Float64("jobs.success_avg_duration_ms", float64(success_duration/successes)))
	span.SetAttributes(attribute.Int64("jobs.failures", int64(failures)))

	span.End()
	defer func() { _ = tp.Shutdown(ctx) }()
	time.Sleep(2 * time.Second)
}
