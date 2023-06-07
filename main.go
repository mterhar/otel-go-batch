package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	jobIdentifier := String(12)
	// intialization code that starts up, does evaluations, connects to queue
	log.Println("starting scheduler: " + jobIdentifier)

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
	// Since we are going to manually kill this span, I don't know if we want to do the normal defer
	// If the app is killed, it won't send.
	// defer span.End()

	span.SetAttributes(attribute.String("job.run", jobIdentifier))

	var childSpan trace.Span
	ctx, childSpan = tracer.Start(ctx, "Child span for evaluating the queue")
	childSpan.SetAttributes(attribute.Int("queue.depth", 1000))

	var grandChildSpan trace.Span
	ctx, grandChildSpan = tracer.Start(ctx, "Grandchild span reporting no errors")
	grandChildSpan.SetAttributes(attribute.Bool("errors", false))

	grandChildSpan.End()
	childSpan.End()
	// startupTraceSpanLink := trace.LinkFromContext(ctx, ))
	startupTraceSpanLink := trace.Link{
		SpanContext: span.SpanContext(),
		Attributes: []attribute.KeyValue{
			attribute.String("name", "Link to job start"),
			attribute.String("job.run", jobIdentifier),
		},
	}
	fmt.Printf("Startup trace span link: %#v \n", startupTraceSpanLink)
	span.End()

	_ = tp.Shutdown(ctx)
	// we now have no active spans on that first trace.
	// the root span was sent so the job is started and a sampling decision can be made.
	// typically the last span comes out when the app ends, but in our case, we don't want that
	// Going forward, we want to treat the spans like metrics, so we'll create a new tracer for that.
	ctxWorker, err := initWorkerTracer()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = tpWorker.Shutdown(ctxWorker) }()

	var spanWorker trace.Span
	lastNewTrace := time.Now()
	ctxWorker, spanWorker = tpWorker.Tracer("example/otel-go-batch").Start(ctxWorker, "First unit of jobs started", trace.WithLinks(startupTraceSpanLink))
	fmt.Printf("first worker-associated span: %#v \n", spanWorker)
	defer spanWorker.End()
	// This for loop is our fake job queue.
	var i = 1
	for ; i <= 10; i++ {
		fmt.Printf("new trace at jobnumber %d and time %s", i, lastNewTrace.String())
		ctxWorker, spanWorker = tpWorker.Tracer("example/otel-go-batch").Start(context.Background(), "Next unit of jobs started", trace.WithLinks(startupTraceSpanLink))

		spanWorker.SetAttributes(attribute.Int("job.number", i))
		err := doSomeLengthyJobWork(ctxWorker, int64(i))
		if err != nil {
			spanWorker.SetStatus(codes.Error, fmt.Sprintf("An error during lengthy job %v", i))
		}
		spanWorker.End()
	}
	// now that all the manual shutdown and restart stuff is done, let's let this tracer die pleasantly
	defer func() { _ = tpWorker.Shutdown(ctxWorker) }()
	spanWorker.SetAttributes(attribute.Bool("job.is_last", true))
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
// allows different sampler, different batching, etc.
// we start it in the scheduler to pass context to the worker rather than the worker starting its own traces
func initWorkerTracer() (context.Context, error) {
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

// helper functions unrelated to otel

// Make a random string for job identifier
// from https://www.calhoun.io/creating-random-strings-in-go/
const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}
