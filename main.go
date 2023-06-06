package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func main() {

	// intialization code that starts up, does evaluations, connects to queue
	log.Println("starting scheduler")

	// create regular tracer for the scheduler
	ctx, err := initTracer()
	if err != nil {
		log.Panic(err)
	}
	tracer := tp.Tracer("example/otel-go-batch")
	// defer func() { _ = tp.Shutdown(ctx) }()

	// make a little tree of spans
	var span trace.Span
	ctx, span = tracer.Start(ctx, "Evaluate the queue and environment")
	// defer span.End()

	span.SetAttributes(attribute.Int("job.run", 2001))

	var childSpan trace.Span
	ctx, childSpan = tracer.Start(ctx, "Child span for evaluating the queue")
	childSpan.SetAttributes(attribute.Int("queue.depth", 1000))

	var grandChildSpan trace.Span
	ctx, grandChildSpan = tracer.Start(ctx, "Grandchild span reporting no errors")
	grandChildSpan.SetAttributes(attribute.Bool("errors", false))

	grandChildSpan.End()
	childSpan.End()
	span.End()

	_ = tp.Shutdown(ctx)
	// we now have no active spans on that first trace.
	// the root span was sent so the job is started and a sampling decision can be made.
	// typically the last span comes out when the app ends, but in our case, we don't want that
	// Going forward, we want to treat the spans like metrics, so we'll create a new tracer for that.
	ctxWorker, err := newWorkerTracer()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = tpWorker.Shutdown(ctxWorker) }()

	var spanWorker trace.Span
	failures := 0
	successes := 0
	lastNewTrace := time.Now()
	ctxWorker, spanWorker = tpWorker.Tracer("example/otel-go-batch").Start(ctxWorker, "First unit of jobs started")
	defer spanWorker.End()
	// This for loop is our fake job queue.
	var i = 1
	for ; i <= 1000; i++ {
		// fmt.Printf("There last timestamp is %d and the current time is %d and restart variable is %b", lastNewTrace, time.Now(), lastNewTrace.Add(30*time.Second).Before(time.Now()))
		// check for a number of iterations, make a new context and spanworker.
		if i%100 == 0 || lastNewTrace.Add(30*time.Second).Before(time.Now()) {
			fmt.Printf("new trace at jobnumber %d and time %s", i, lastNewTrace.String())

			spanWorker.SetAttributes(attribute.Int("job.period.ending_number", i-1))
			spanWorker.SetAttributes(attribute.Int("job.period.failures", failures))
			failures = 0
			spanWorker.SetAttributes(attribute.Int("job.period.successes", successes))
			successes = 0
			lastNewTrace = time.Now()

			spanWorker.End()
			// need to make a new variable?
			// reassign the context to a new fresh one and make a new spanWorker.
			ctxWorker, spanWorker = tpWorker.Tracer("example/otel-go-batch").Start(context.Background(), "Next unit of jobs started")

			spanWorker.SetAttributes(attribute.Int("job.starting_number", i))
		}

		err := doSomeJobWork(ctxWorker, int64(i))
		if err != nil {
			failures += 1
		}
		successes += 1
	}
	spanWorker.SetAttributes(attribute.Int("job.period.ending_number", i-1))
	spanWorker.SetAttributes(attribute.Int("job.period.failures", failures))
	spanWorker.SetAttributes(attribute.Int("job.period.successes", successes))
	spanWorker.End()
	// now that all the manual shutdown and restart stuff is done, let's let this tracer die pleasantly
	defer func() { _ = tpWorker.Shutdown(ctxWorker) }()

	// do some job cleanup stuff?
	log.Println("Done with the batch jobs")
}

// This section is for the scheduler's tracer.
var tp *sdktrace.TracerProvider

// initTracer creates and registers trace provider instance.
func initTracer() (context.Context, error) {
	ctx := context.Background()
	client := otlptracegrpc.NewClient()
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize grpc exporter: %w", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tp)
	return ctx, nil
}

var tpWorker *sdktrace.TracerProvider

// the next function is to create a new trace proider for the worker.
func newWorkerTracer() (context.Context, error) {
	// replace with honeycomb exporter?
	ctx := context.Background()
	client := otlptracegrpc.NewClient()
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize grpc exporter: %w", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tpWorker = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
	)
	return ctx, nil
}
