import fastapi
import asyncio
import requests
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry import trace
from opentelemetry.instrumentation.requests import RequestsInstrumentor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import (
    ConsoleSpanExporter,
    SimpleSpanProcessor,
    BatchSpanProcessor,
)

def initOTEL(app):
    otlp_exporter = OTLPSpanExporter(endpoint="http://jaeger:4317", insecure=True)
    span_processor = BatchSpanProcessor(otlp_exporter)
    trace.set_tracer_provider(TracerProvider())
    tracer = trace.get_tracer_provider().get_tracer(__name__)
    trace.get_tracer_provider().add_span_processor(span_processor)
    # RequestsInstrumentor().instrument()  # Uncoment this line to more MAGIC!
    FastAPIInstrumentor.instrument_app(app)


app = fastapi.FastAPI()
initOTEL(app)

@app.get("/foobar")
async def foobar():
    return {"message": "hello world"}

@app.get("/sum/{a}/{b}")
async def sum(a, b):
    a = int(a)
    b = int(b)
    for i in range(a):
        # Do another request to add even more colors to our tracer
        r = requests.post("http://httpbin:80/anything", {"foo":"Bar"})
        await asyncio.sleep(b/1000)
    total = a + b
    return total
