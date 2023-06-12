package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type customErrorClump struct {
	SchedulingFailures        int
	RecoverableStartupFailure int
	ConnectionFailure         int
	TooManyPuppies            int
}

func optionalTrace() {

	ctx, err := initTracer()
	if err != nil {
		log.Panic(err)
	}
	tracer := tp.Tracer("example/otel-go-batch")
	// defer func() { _ = tp.Shutdown(ctx) }()

	// make a little tree of spans
	var span trace.Span
	_, span = tracer.Start(ctx, "Evaluate the queue and environment")
	defer span.End()

	startupTraceSpanLink := trace.Link{
		SpanContext: span.SpanContext(),
		Attributes: []attribute.KeyValue{
			attribute.String("name", "Link to job start"),
		},
	}

	// ending the initialization span.
	span.End()
	jobs := make([]int, 1000)
	// errorsPile is a set of errors where the index is the job id and then each custom error is part of the clump
	errorsPile := make(map[int]customErrorClump)
	var successes int64
	for i := range jobs {
		// do the jobs and collect up the errors
		errIntArray := doSomeDetailedJobWork2(int64(i))
		if errIntArray != nil {
			// store the errors in the pile for later summarization
			errorsPile[i] = customErrorClump{
				SchedulingFailures:        errIntArray[0],
				RecoverableStartupFailure: errIntArray[1],
				ConnectionFailure:         errIntArray[2],
				TooManyPuppies:            errIntArray[3],
			}
		} else {
			successes += 1
		}
	}

	time.Sleep(2 * time.Second)

	summaryErrorClump := customErrorClump{}
	var failedJobsIds []string
	// analyze the errors
	for i, cec := range errorsPile {
		summaryErrorClump.SchedulingFailures += cec.SchedulingFailures
		summaryErrorClump.RecoverableStartupFailure += cec.RecoverableStartupFailure
		summaryErrorClump.ConnectionFailure += cec.ConnectionFailure
		summaryErrorClump.TooManyPuppies += cec.TooManyPuppies

		failedJobsIds = append(failedJobsIds, fmt.Sprintf("%v", i))
	}

	// create the summary trace (single span)
	var spanSummary trace.Span
	ctx, spanSummary = tracer.Start(context.Background(), "End of batch run summary", trace.WithLinks(startupTraceSpanLink))
	spanSummary.SetAttributes(attribute.String("job.emitted_by", "scheduler"))
	spanSummary.SetAttributes(attribute.String("summary.failed_job.ids", strings.Join(failedJobsIds, ", ")))
	spanSummary.SetAttributes(attribute.Int64("summary.failed.count", int64(len(errorsPile))))
	spanSummary.SetAttributes(attribute.Int64("Summary.success.count", successes))

	spanSummary.SetAttributes(attribute.Int64("summary.error.SchedulingFailures.count", int64(summaryErrorClump.SchedulingFailures)))
	spanSummary.SetAttributes(attribute.Int64("summary.error.RecoverableStartupFailure.count", int64(summaryErrorClump.RecoverableStartupFailure)))
	spanSummary.SetAttributes(attribute.Int64("summary.error.ConnectionFailure.count", int64(summaryErrorClump.ConnectionFailure)))
	spanSummary.SetAttributes(attribute.Int64("summary.error.TooManyPuppies.count", int64(summaryErrorClump.TooManyPuppies)))

	spanSummary.End()

	time.Sleep(50 * time.Millisecond)

	defer func() { _ = tp.Shutdown(ctx) }()
	time.Sleep(2 * time.Second)
}
