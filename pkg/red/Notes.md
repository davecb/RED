# Notes on RED

## Trivial-test results:
```shell
/usr/local/go/bin/go build -o /tmp/GoLand/___go_build_model_uploader . #gosetup
/tmp/GoLand/___go_build_model_uploader -config conf/local-test-model-uploader.toml
```
```json
{"level":"INFO","timestamp":"2021-12-24T09:42:26.422-0500","caller":"model-uploader/main.go:29","message":"Successfully initialized logger","version":"","PID":9450}
{"level":"INFO","timestamp":"2021-12-24T09:42:27.101-0500","caller":"server/server.go:45","message":"Successfully initialized aerospike","version":"","PID":9450}
{"level":"INFO","timestamp":"2021-12-24T09:42:27.101-0500","caller":"server/server.go:52","message":"Successfully initialized sftp","version":"","PID":9450}
{"level":"INFO","timestamp":"2021-12-24T09:42:27.102-0500","caller":"server/server.go:128","message":"profiling (pprof) is enabled at /internal/profiling","version":"","PID":9450}
{"level":"INFO","timestamp":"2021-12-24T09:42:27.102-0500","caller":"server/server.go:80","message":"http server listening at address '0.0.0.0:7723'","version":"","PID":9450}
{"level":"INFO","timestamp":"2021-12-24T09:42:47.055-0500","caller":"server/experiment.go:36","message":"received an upload request for 'BTO_experiment_NY'","version":"","PID":9450}
{"level":"INFO","timestamp":"2021-12-24T10:08:55.164-0500","caller":"server/experiment.go:46","message":"successfully uploaded","version":"","PID":9450,"set":"BTO_experiment_NY","red":"45225, 0, 0.000000s"}
^C
{"level":"INFO","timestamp":"2021-12-24T12:45:22.003-0500","caller":"model-uploader/main.go:91","message":"Received signal 'interrupt', stopping server","version":"","PID":9450}
{"level":"INFO","timestamp":"2021-12-24T12:45:22.004-0500","caller":"server/server.go:85","message":"http server stopped: http: Server closed","version":"","PID":9450}
{"level":"INFO","timestamp":"2021-12-24T12:45:22.005-0500","caller":"model-uploader/main.go:95","message":"model uploader service stopped","version":"","PID":9450}

```
That was from home, and 
* the format is ugly
* the duration is wrong

The duration should have been 10:08:55.164 - 09:42:47.055 = 26:08.11

Second run fixed the duration, got even uglier
```json
{"level":"INFO","timestamp":"2021-12-24T15:09:18.789-0500","caller":"server/experiment.go:45","message":"successfully uploaded 'BTO_experiment_NY', red = 45225, 0, 1570.586371s","version":"","PID":121009},
{"level":"INFO","timestamp":"2021-12-24T15:09:18.789-0500","caller":"server/experiment.go:47","message":"successfully uploaded set","version":"","PID":121009,"set":"BTO_experiment_NY","red":"eyJyZXF1ZXN0cyI6NDUyMjUsImVycm9ycyI6MCwiZHVyYXRpb24iOjE1NzA1ODY0NzE4MTV9"}

```
Third gets a little closer, but duration is a time.Duration
```json
{"level":"INFO","timestamp":"2021-12-24T15:29:22.807-0500","caller":"server/experiment.go:36","message":"received an upload request for 'BTO_experiment_NY'","version":"","PID":128602},
{"level":"INFO","timestamp":"2021-12-24T15:55:38.770-0500","caller":"server/experiment.go:45","message":"successfully uploaded 'BTO_experiment_NY'","version":"","PID":128602},
{"level":"INFO","timestamp":"2021-12-24T15:55:38.770-0500","caller":"server/experiment.go:47","message":"upload metrics","version":"","PID":128602,"set":"BTO_experiment_NY","requests":45225,"errors":0,"duration":1575963193960},
```
And the fourth looks vaguely sane
```json
{"level":"INFO","timestamp":"2021-12-24T17:07:39.925-0500","caller":"server/experiment.go:36","message":"received an upload request for 'BTO_experiment_NY'","version":"","PID":153382},
{"level":"INFO","timestamp":"2021-12-24T17:33:43.487-0500","caller":"server/experiment.go:45","message":"successfully uploaded 'BTO_experiment_NY'","version":"","PID":153382},
{"level":"INFO","timestamp":"2021-12-24T17:33:43.487-0500","caller":"server/experiment.go:47","message":"upload metrics","version":"","PID":153382,"set":"BTO_experiment_NY","requests":45225,"errors":0,"duration":1563.562311863},

```

# The interface used by kludgy one is
metrics.Record(ctx, metricConfusionMatrixTruePositive, 1)

in echange-node/telemetry/context.go, it's 
```
// Record will record a metric having access to any tags inherited from the
// context or set explicitly in the tags argument.
func Record(ctx context.Context, name string, i float64, tags ...Tags) error {
	if atomic.LoadInt32(&recordingEnabled) == 0 && !criticalMetrics[name] {
		return nil
	}

	metricLock.RLock()
	defer metricLock.RUnlock()

	metric, ok := metricCache[name]
	if !ok {
		return fmt.Errorf("no registered metric of name: %s", name)
	}

	if len(tags) == 1 {
		mutators := make([]tag.Mutator, 0, len(tags[0]))

		for tagName, tagValue := range tags[0] {
			theTag, ok := metricTags[tagName]
			if !ok {
				return fmt.Errorf("tag is not in list of provided tags")
			}

			mutators = append(mutators, tag.Upsert(theTag, tagValue))
		}

		ctx, _ = tag.New(ctx, mutators...)
	}

	stats.Record(ctx, metric.Measure(i))

	return nil
}

which calls opencensus Record -> RecordWithOptions -> recorder, opaquely


