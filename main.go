package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	jobIdentifier := String(12)
	// intialization code that starts up, does evaluations, connects to queue
	log.Println("starting scheduler: " + jobIdentifier)

	optionalTrace()

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
