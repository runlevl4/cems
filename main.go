package main

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MichaelS11/go-dht"
	"github.com/ardanlabs/conf"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
)

var build = "develop"
var applog = log.New(os.Stdout, "CEMS : ", log.LstdFlags|log.Lmicroseconds)

func main() {
	log := log.New(os.Stdout, "CEMS : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	if err := run(log); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func run(log *log.Logger) error {
	// =========================================================================
	// Configuration

	var cfg struct {
		conf.Version
		Web struct {
			APIHost         string        `conf:"default:0.0.0.0:3500"`
			DebugHost       string        `conf:"default:0.0.0.0:4000"`
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:5s"`
			ShutdownTimeout time.Duration `conf:"default:5s,noprint"`
		}
		App struct {
			PollSecs time.Duration `conf:"default:5s"`
		}
	}

	cfg.Version.SVN = build
	cfg.Version.Desc = "Copyright (c) 2020 runlevl4)"

	if err := conf.Parse(os.Args[1:], "CEMS", &cfg); err != nil {
		switch err {
		case conf.ErrHelpWanted:
			usage, err := conf.Usage("CEMS", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config usage")
			}
			fmt.Println(usage)
			return nil
		case conf.ErrVersionWanted:
			version, err := conf.VersionString("CEMS", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config version")
			}
			fmt.Println(version)
			return nil
		}
		return errors.Wrap(err, "parsing config")
	}

	// =========================================================================
	// App Starting

	// Print the build version for our logs. Also expose it under /debug/vars.
	expvar.NewString("build").Set(build)
	log.Printf("main : Started : Application initializing : version %q", build)
	defer log.Println("main: Completed")

	out, err := conf.String(&cfg)
	if err != nil {
		return errors.Wrap(err, "generating config for output")
	}
	log.Printf("main: Config :\n%v\n", out)

	// =========================================================================
	// Start Debug Service
	//
	// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
	// /debug/vars - Added to the default mux by importing the expvar package.

	log.Println("main: Initializing debugging support")

	go func() {
		log.Printf("main: Debug Listening %s", cfg.Web.DebugHost)
		if err := http.ListenAndServe(cfg.Web.DebugHost, http.DefaultServeMux); err != nil {
			log.Printf("main: Debug Listener closed : %v", err)
		}
	}()

	// go getStats(applog, cfg.App.PollSecs)
	// go checkWaterLevel(applog, cfg.App.PollSecs)
	go captureStats(applog, cfg.App.PollSecs)

	// Start metrics server for Prometheus.
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Printf("main: Metrics listening on %s", "2112")
		http.ListenAndServe(":2112", nil)
	}()

	// =========================================================================
	// Start API Service

	log.Println("main: Initializing API support")

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	mux := http.NewServeMux()
	mux.HandleFunc("/", stats)

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      mux,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
	}

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		log.Printf("main: API listening on %s", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// =========================================================================
	// Shutdown

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return errors.Wrap(err, "server error")

	case sig := <-shutdown:
		log.Printf("main: %v : Start shutdown", sig)

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		// Asking listener to shutdown and shed load.
		if err := api.Shutdown(ctx); err != nil {
			api.Close()
			return errors.Wrap(err, "could not stop server gracefully")
		}
	}

	return nil
}

func stats(w http.ResponseWriter, r *http.Request) {
	err := dht.HostInit()
	if err != nil {
		fmt.Println("HostInit error:", err)
		//return err
	}

	dht, err := dht.NewDHT("GPIO4", dht.Fahrenheit, "")
	if err != nil {
		fmt.Println("NewDHT error:", err)
		//return err
	}

	humidity, temperature, err := dht.ReadRetry(11)
	if err != nil {
		fmt.Println("Read error:", err)
		//return err
	}

	stats := struct {
		Humidity    float64 `json:"humidity"`
		Temperature float64 `json:"temperature"`
	}{
		Humidity:    humidity,
		Temperature: temperature,
	}

	b, err := json.Marshal(stats)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"%s"\n`, err)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, string(b))
}

func getStats(log *log.Logger, pollSecs time.Duration) {
	// go func() {
	for {
		err := dht.HostInit()
		if err != nil {
			log.Println("HostInit error:", err)
		}

		dht, err := dht.NewDHT("GPIO4", dht.Fahrenheit, "")
		if err != nil {
			log.Println("NewDHT error:", err)
		}

		humidity, temperature, err := dht.ReadRetry(11)
		if err != nil {
			log.Println("Read error:", err)
		}

		stats := struct {
			Humidity    float64 `json:"humidity"`
			Temperature float64 `json:"temperature"`
		}{
			Humidity:    humidity,
			Temperature: temperature,
		}

		b, err := json.Marshal(stats)
		if err != nil {
			log.Printf("error parsing response | %s", err)
		}

		dhTemp.Set(temperature)
		dhHumidity.Set(humidity)

		log.Println(string(b))

		time.Sleep(pollSecs)
	}
	// }()

}

func checkWaterLevel(log *log.Logger, pollSecs time.Duration) {
	// go func() {
	// Setup pins.
	host.Init()
	trigger := gpioreg.ByName("GPIO23")
	echo := gpioreg.ByName("GPIO24")

	for {
		// Initiate trigger.
		trigger.Out(gpio.High)
		time.Sleep(time.Microsecond * 10)
		trigger.Out(gpio.Low)

		// Evaluate echo and calculate distance.
		var startTime, endTime int64
		for echo.Read() == gpio.Low {
			startTime = time.Now().UnixNano()
		}
		for echo.Read() == gpio.High {
			endTime = time.Now().UnixNano()
		}
		distance := (float32(endTime-startTime) * 17150) / 1e9
		log.Printf("distance: %f\n", distance/2.54)

		dhWaterLevel.Set(float64((distance / 2.54) * -1))

		dist := struct {
			Distance float64
		}{
			Distance: float64(distance / 2.54),
		}

		b, err := json.Marshal(dist)
		if err != nil {
			log.Printf("error parsing distance | %s", err)
		}
		log.Printf(string(b))

		time.Sleep(pollSecs)
	}

	// }()

}

func captureStats(log *log.Logger, pollSecs time.Duration) {
	// fmt.Println("pollSecs: ", pollSecs)
	// go func() {
	for {
		// Check temp/humidity.
		err := dht.HostInit()
		if err != nil {
			log.Println("HostInit error:", err)
		}

		dht, err := dht.NewDHT("GPIO4", dht.Fahrenheit, "")
		if err != nil {
			log.Println("NewDHT error:", err)
		}

		humidity, temperature, err := dht.ReadRetry(11)
		if err != nil {
			log.Println("Read error:", err)
		}

		// Check water level.
		host.Init()
		trigger := gpioreg.ByName("GPIO23")
		echo := gpioreg.ByName("GPIO24")

		// Initiate trigger.
		trigger.Out(gpio.High)
		time.Sleep(time.Microsecond * 10)
		trigger.Out(gpio.Low)

		// Evaluate echo and calculate distance.
		var startTime, endTime int64
		for echo.Read() == gpio.Low {
			startTime = time.Now().UnixNano()
		}
		for echo.Read() == gpio.High {
			endTime = time.Now().UnixNano()
		}
		distance := (float32(endTime-startTime) * 17150) / 1e9
		// log.Printf("distance: %f\n", distance/2.54)

		dhWaterLevel.Set(float64((distance / 2.54) * -1))

		// Set metrics for Prometheus.
		dhTemp.Set(temperature)
		dhHumidity.Set(humidity)
		dhWaterLevel.Set(float64((distance / 2.54) * -1))

		// Build response structure and marshal to JSON.
		stats := struct {
			Humidity    float64 `json:"humidity"`
			Temperature float64 `json:"temperature"`
			WaterLevel  float64 `json:"waterLevel"`
		}{
			Humidity:    humidity,
			Temperature: temperature,
			WaterLevel:  float64(distance / 2.54),
		}

		b, err := json.Marshal(stats)
		if err != nil {
			log.Printf("error parsing response | %s", err)
		}

		// Log to console.
		log.Printf(string(b))

		// Take a break.
		time.Sleep(pollSecs)
	}
	// }()
}

var (
	dhTemp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cems_temperature",
		Help: "The current temperature.",
	})
	dhHumidity = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cems_humidity",
		Help: "The current humidity level.",
	})
	dhWaterLevel = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cems_water_level",
		Help: "The current water level measured in cm.",
	})
)

func sendEmail() error {
	pwd := os.Getenv("CEMS_SMTP_PWD")
	if pwd == "" {
		return errors.New("missing smtp password")
	}

	smtpHost := "mail.runlevl4.com:2525"
	to := []string{"root@runlevl4.com"}
	from := "cems@runlevl4.com"

	conn, err := net.DialTimeout("tcp", smtpHost, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	
	auth := sasl.NewPlainClient("", from, pwd)

	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	msg := strings.NewReader("To: root@runlevl4.com\r\n" +
		"Subject: CEMS Warning\r\n" +
		"\r\n" +
		"It's time to refill the misting tank.\r\n")
	err = smtp.SendMail(smtpHost, auth, from, to, msg)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
