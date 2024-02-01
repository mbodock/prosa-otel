package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var tracer = otel.Tracer("gin-server")

func main() {
	tp, err := initTracer()
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	r := gin.Default()
	r.Use(otelgin.Middleware("Golang service middleware"))
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/sum", func(c *gin.Context) {
		otel.GetTextMapPropagator().Inject(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		a, err := strconv.Atoi(c.PostForm("a"))
		b, err := strconv.Atoi(c.PostForm("b"))
		if err != nil {
			c.JSON(400, gin.H{"msg": err})
			return
		}

		result := fetchSumResult(c.Request.Context(), a, b)
		c.JSON(http.StatusOK, gin.H{
			"result": fmt.Sprintf("%d", result),
		})
	})
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func fetchSumResult(ctx context.Context, a, b int) int {
	traceID := "fetchSum"
	tracer := otel.GetTracerProvider().Tracer("")
	ctx, span := tracer.Start(ctx, traceID,
		trace.WithAttributes(attribute.String("args", fmt.Sprintf("a=%d,b=%d", a, b))),
	)
	defer span.End()

	url := fmt.Sprintf("http://python:8081/sum/%d/%d", a, b)
	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	tCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(tCtx, http.MethodGet, url, nil)
	resp, _ := client.Do(req)

	data, _ := io.ReadAll(resp.Body)
	sum, _ := strconv.Atoi(string(data))
	span.AddEvent("response code", trace.WithAttributes(attribute.Int("code", resp.StatusCode)))

	var meter = otel.Meter(traceID)
	apiCounter, err := meter.Int64Counter(
		"sumApiUses",
		metric.WithDescription("Number of times we used the sum api."),
		metric.WithUnit("{call}"),
	)
	apiCounter.Add(tCtx, 1)
	if err != nil {
		panic(err)
	}
	return sum
}

func initTracer() (*sdktrace.TracerProvider, error) {
	// Set up tracer exporter
	ctx := context.Background()
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint("jaeger:4317"), // See github.com/moby/moby/issues/46129
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", "Golang service"),
			attribute.String("service.namespace", "prosa.golang.service"),
			attribute.String("service.instance.id", os.Getenv("HOSTNAME")),
		),
	)
	if err != nil {
		panic(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set up metrics provider prometheus.
	// DefaultRegisterer is used by default
	// so that metrics are available via promhttp.Handler.
	mexporter, err := prometheus.New()
	if err != nil {
		panic(err)
	}
	pprovider := sdkMetric.NewMeterProvider(sdkMetric.WithReader(mexporter))
	otel.SetMeterProvider(pprovider)                      // Sets global metrics provider
	otel.SetTracerProvider(tp)                            // Sets global tracer provider
	otel.SetTextMapPropagator(propagation.TraceContext{}) // Sets global propagator provider

	return tp, nil
}
