package main

import (
  "example.com/m/v2/util"
  "example.com/m/v2/worker/tasks"
  "github.com/hibiken/asynq"
  "context"
  "github.com/tidwall/gjson"
  "example.com/m/v2/store"
  "fmt"
  "encoding/json"
  "strings"
  "os"
    "os/signal"
  "net/http"
   "io/ioutil"
  // "io/ioutil"
  "bytes"
  "time"
  "errors"
  "github.com/gocolly/colly/v2"
  "net/url"
  "github.com/prometheus/client_golang/prometheus"
  "github.com/prometheus/client_golang/prometheus/promauto"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "golang.org/x/sys/unix"
)

var s store.Store

var METADATA_HOST = util.GetEnv("METADATA_HOST", "http://127.0.0.1:1969/search")
var SEARCH_HOST = util.GetEnv("SEARCH_HOST", "https://www.google.com")
// Metrics variables.
var (
    processedCounter = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "processed_tasks_total",
            Help: "The total number of processed tasks",
        },
        []string{"task_type"},
    )

    failedCounter = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "failed_tasks_total",
	    Help: "The total number of times processing failed",
	},
        []string{"task_type"},
    )

    inProgressGauge = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
	    Name: "in_progress_tasks",
	    Help: "The number of tasks currently being processed",
	},
        []string{"task_type"},
    )
)

func metricsMiddleware(next asynq.Handler) asynq.Handler {
    return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
        inProgressGauge.WithLabelValues(t.Type()).Inc()
        err := next.ProcessTask(ctx, t)
        inProgressGauge.WithLabelValues(t.Type()).Dec()
	if err != nil {
	    failedCounter.WithLabelValues(t.Type()).Inc()
	}
	processedCounter.WithLabelValues(t.Type()).Inc()
	return err
    })
}

func main() {
  var storeErr error
  s, storeErr = store.NewStore(&store.CreateStoreOptions{
    EnableCache: true,
    ReadOnlyOrbit: false,
  })
  if storeErr != nil {
    panic(storeErr)
  }
  httpServeMux := http.NewServeMux()
      httpServeMux.Handle("/metrics", promhttp.Handler())
      metricsSrv := &http.Server{
          Addr:    ":2112",
  	      Handler: httpServeMux,
      }
      done := make(chan struct{})

      // Start metrics server.
      go func() {
          err := metricsSrv.ListenAndServe()
  	if err != nil && err != http.ErrServerClosed {
  	    fmt.Printf("Error: metrics server error: %v", err)
  	}
  	close(done)
      }()
    srv := asynq.NewServer(tasks.NewAsyncRedisConnection(),asynq.Config{
      Concurrency: 1,
    })
    mux := asynq.NewServeMux()
    mux.Use(metricsMiddleware)
    mux.HandleFunc(tasks.TypeDocFetch, HandleDocFetchTask)
    if err := srv.Start(mux); err != nil {
        fmt.Println(err)
    }
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, unix.SIGTERM, unix.SIGINT, unix.SIGTSTP)
    for {
        s := <-sigs
        if s == unix.SIGTSTP {
	    srv.Stop() // Stop processing new tasks
	    continue
	}
	break
    }

    // Stop worker server.
    srv.Shutdown()

    // Stop metrics server.
    if err := metricsSrv.Shutdown(context.Background()); err != nil {
        fmt.Printf("Error: metrics server shutdown error: %v", err)
    }
    <-done
}

func FetchMetadata(docId string) (string, error) {
  req, _ := http.NewRequest("POST", METADATA_HOST, bytes.NewBuffer([]byte(docId)))
  req.Header.Set("Content-Type", "text/plain")
	client := &http.Client{
		Timeout:   10 * time.Second,
	}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", errors.New("Received non 200 response code for metadata for " + docId)
	}
  resBytes, err := ioutil.ReadAll(response.Body)
    if err != nil {
      return "", err
    }
  value := gjson.Get(string(resBytes), "#(title).title")
	return value.String(), nil
}

func HandleDocFetchTask(ctx context.Context, t *asynq.Task) error {
    var p tasks.DocFetchPayload
    if err := json.Unmarshal(t.Payload(), &p); err != nil {
        return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
    }
    docId := p.DocId

    // get metadata
    content, err := FetchMetadata(docId)
    fmt.Println(content)
    if err != nil {
      fmt.Println(err)
      return err
    }

    documentUrl, err := getAndParsePage(docId)
    if err != nil {
      fmt.Println(err)
      return err
    }
    fileBytes, err := downloadFile(documentUrl)
    if err != nil {
      fmt.Println(err)
      return err
    }
    // todo add fileBytes to ipfs (no pin)
    ipfsRef, err := s.SaveDocument(ctx, bytes.NewReader(fileBytes))
    if err != nil {
      fmt.Println(err)
  		return err
  	}
    fmt.Println(ipfsRef)

    // PUT into db
    err = s.Put(store.Entry{
      Id: docId,
      Content: content,
      IpfsRef: ipfsRef,
      }, ctx)
    if err != nil {
      fmt.Println(err)
      return err
    }
    return nil
}

func downloadFile(baseUrl string) ([]byte, error) {
  var fileUrl string
  if !strings.HasPrefix(baseUrl, "http"){
    fileUrl = "https://" + baseUrl
  } else {
    fileUrl = baseUrl
  }
  req, _ := http.NewRequest("GET", fileUrl, nil)
	client := &http.Client{
		Timeout:   30 * time.Second,
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, errors.New("Received non 200 response code")
	}
  fileBytes, err := ioutil.ReadAll(response.Body)
  if err != nil {
		return nil, err
	}
	return fileBytes, nil
}

// given a url with search parameters,
// returns the url of the document found on the response page
func getAndParsePage(docId string) (string, error) {

	c := colly.NewCollector()
  var res string
  var resErr error
  // grab the src of the iframe contained within a div that has id=article
	c.OnHTML("div[id=article]", func(e *colly.HTMLElement) {
		docUrl := e.ChildAttr("iframe", "src")
    parsed, err := url.Parse(docUrl)
    if err != nil {
      resErr = err
    } else {
      res = parsed.Host + parsed.Path
    }
	})
	c.OnError(func(r *colly.Response, err error) {
    resErr = err
	})

	c.Post(SEARCH_HOST, map[string]string{"request": docId})

  return res, resErr
}
