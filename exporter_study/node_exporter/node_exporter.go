package main

import (
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/user"
	"sort"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	promcollectors "github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"      // 定义了prometheus日志初始化格式
	"github.com/prometheus/common/promlog/flag" // 命令行参数解析工具
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"

	// "github.com/prometheus/node_exporter/collector" // 具体收集功能代码
	"node_exporter/collector"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// handler 实现了http包中Handler的ServeHTTP方法

/*
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}
*/
type handler struct {
	unfilteredHandler http.Handler
	// exporterMetricsRegistry is a separate registry for the metrics about
	// the exporter itself.
	exporterMetricsRegistry *prometheus.Registry
	includeExporterMetrics  bool
	maxRequests             int
	logger                  log.Logger
}

func newHandler(includeExporterMetrics bool, maxRequests int, logger log.Logger) *handler {
	h := &handler{
		exporterMetricsRegistry: prometheus.NewRegistry(),
		includeExporterMetrics:  includeExporterMetrics,
		maxRequests:             maxRequests,
		logger:                  logger,
	}

	if h.includeExporterMetrics {
		h.exporterMetricsRegistry.MustRegister(
			promcollectors.NewProcessCollector(promcollectors.ProcessCollectorOpts{}),
			promcollectors.NewGoCollector(),
		)
	}
	if innerHandler, err := h.innerHandler(); err != nil {
		panic(fmt.Sprintf("Couldn't create metrics handler: %s", err))
	} else {
		h.unfilteredHandler = innerHandler
	}

	return h

}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filters := r.URL.Query()["collect[]"]
	level.Debug(h.logger).Log("msg", "collect query:", "filters", filters)

	if len(filters) == 0 {
		// No filters, use the prepared unfiltered handler.
		h.unfilteredHandler.ServeHTTP(w, r)
		return
	}
	// To serve filtered metrics, we create a filtering handler on the fly.
	filteredHandler, err := h.innerHandler(filters...)
	if err != nil {
		level.Warn(h.logger).Log("msg", "Couldn't create filtered metrics handler:", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Couldn't create filtered metrics handler: %s", err)))
		return
	}
	filteredHandler.ServeHTTP(w, r)
}

func (h *handler) innerHandler(filters ...string) (http.Handler, error) {
	nc, err := collector.NewNodeCollector(h.logger, filters...)
	if err != nil {
		return nil, fmt.Errorf("couldn't create collector: %s", err)
	}

	// Only log the creation of an unfiltered handler, which should happen
	// only once upon startup.
	if len(filters) == 0 {
		level.Info(h.logger).Log("msg", "Enabled collectors")
		collectors := []string{}
		for n := range nc.Collectors {
			collectors = append(collectors, n)
		}
		sort.Strings(collectors)
		for _, c := range collectors {
			level.Info(h.logger).Log("collector", c)
		}
	}

	r := prometheus.NewRegistry()
	r.MustRegister(version.NewCollector("node_exporter"))
	if err := r.Register(nc); err != nil {
		return nil, fmt.Errorf("couldn't register node collector: %s", err)
	}
	handler := promhttp.HandlerFor(
		prometheus.Gatherers{h.exporterMetricsRegistry, r},
		promhttp.HandlerOpts{
			ErrorLog:            stdlog.New(log.NewStdlibAdapter(level.Error(h.logger)), "", 0),
			ErrorHandling:       promhttp.ContinueOnError,
			MaxRequestsInFlight: h.maxRequests,
			Registry:            h.exporterMetricsRegistry,
		},
	)
	if h.includeExporterMetrics {
		// Note that we have to use h.exporterMetricsRegistry here to
		// use the same promhttp metrics for all expositions.
		handler = promhttp.InstrumentMetricHandler(
			h.exporterMetricsRegistry, handler,
		)
	}
	return handler, nil
}

func main() {

	// 返回指定flag参数，不指定使用默认值
	var (
		listenAddress = kingpin.Flag(
			"web.listen-address",
			"Address on which to expose metrics and web interface.",
		).Default(":9100").String()
		metricsPath = kingpin.Flag(
			"web.telemetry-path",
			"Path under which to expose metrics.",
		).Default("/metrics").String()
		disableExporterMetrics = kingpin.Flag(
			"web.disable-exporter-metrics",
			"Exclude metrics about the exporter itself (promhttp_*, process_*, go_*).",
		).Bool()
		maxRequests = kingpin.Flag(
			"web.max-requests",
			"Maximum number of parallel scrape requests. Use 0 to disable.",
		).Default("40").Int()
		disableDefaultCollectors = kingpin.Flag(
			"collector.disable-defaults",
			"Set all collectors to disabled by default.",
		).Default("false").Bool()
		configFile = kingpin.Flag(
			"web.config",
			"[EXPERIMENTAL] Path to config yaml file that can enable TLS or authentication.",
		).Default("").String()
	)

	// fmt.Println(*listenAddress, *metricsPath, *disableExporterMetrics, *maxRequests, *disableDefaultCollectors, *configFile)

	promlogConfig := &promlog.Config{}                //提供 --log.level=info
	flag.AddFlags(kingpin.CommandLine, promlogConfig) //提供 -log.format=logfmt

	kingpin.Version(version.Print("node_exporter"))
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promlog.New(promlogConfig)

	version.BuildUser = "wshile"
	version.BuildDate = "2022.03"

	if *disableDefaultCollectors {
		collector.DisableDefaultCollectors()
	}

	level.Info(logger).Log("msg", "Starting node_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())
	// 判断当前如果是root运行就输出warn日志
	if user, err := user.Current(); err == nil && user.Uid == "0" {
		level.Warn(logger).Log("msg", "Node Exporter is running as root user. This exporter is designed to run as unpriviledged user, root is not required.")
	}

	level.Info(logger).Log("msg", "Listening on", "address", *listenAddress)

	http.Handle(*metricsPath, newHandler(!*disableExporterMetrics, *maxRequests, logger))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Node Exporter</title></head>
			<body>
			<h1>Node Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	server := &http.Server{Addr: *listenAddress}
	if err := web.ListenAndServe(server, *configFile, logger); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}
