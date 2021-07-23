package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"flag"

	"github.com/eze-kiel/uptimerobot-exporter/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
)

type app struct {
	apiKey         string
	address        string
	port           string
	scrapeInterval int
	logLevel       string
	logger         zerolog.Logger
}

type AccountDetails struct {
	Stat    string `json:"stat"`
	Account struct {
		Email                  string    `json:"email"`
		UserID                 int       `json:"user_id"`
		Firstname              string    `json:"firstname"`
		SmsCredits             int       `json:"sms_credits"`
		PaymentProcessor       int       `json:"payment_processor"`
		PaymentPeriod          int       `json:"payment_period"`
		SubscriptionExpiryDate time.Time `json:"subscription_expiry_date"`
		MonitorLimit           int       `json:"monitor_limit"`
		MonitorInterval        int       `json:"monitor_interval"`
		UpMonitors             int       `json:"up_monitors"`
		DownMonitors           int       `json:"down_monitors"`
		PausedMonitors         int       `json:"paused_monitors"`
	} `json:"account"`
}

type MonitorsData struct {
	Stat       string `json:"stat"`
	Pagination struct {
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
		Total  int `json:"total"`
	} `json:"pagination"`
	Monitors []struct {
		ID             int    `json:"id"`
		FriendlyName   string `json:"friendly_name"`
		URL            string `json:"url"`
		Type           int    `json:"type"`
		SubType        string `json:"sub_type"`
		KeywordType    int    `json:"keyword_type"`
		KeywordValue   string `json:"keyword_value"`
		HTTPUsername   string `json:"http_username"`
		HTTPPassword   string `json:"http_password"`
		Port           string `json:"port"`
		Interval       int    `json:"interval"`
		Status         int    `json:"status"`
		CreateDatetime int    `json:"create_datetime"`
		ResponseTimes  []struct {
			Datetime int `json:"datetime"`
			Value    int `json:"value"`
		} `json:"response_times"`
		AverageResponseTime string `json:"average_response_time"`
	} `json:"monitors"`
}

var (
	accountDetails = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "uptimerobot_account_details",
		Help: "Details of the Uptime Robot account",
	}, []string{"firstname", "email", "monitors_limit", "monitor_interval", "up_monitors", "down_monitors", "paused_monitors", "payment_period"})

	upMonitors = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "uptimerobot_up_monitors",
		Help: "Up monitors",
	})

	downMonitors = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "uptimerobot_down_monitors",
		Help: "Down monitors",
	})

	pausedMonitors = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "uptimerobot_paused_monitors",
		Help: "Down monitors",
	})

	monitorsStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "uptimerobot_monitors_status",
		Help: "The total number of processed events",
	}, []string{"url", "friendly_name", "interval"})

	responseTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "uptimerobot_response_time",
		Help: "Monitors response times",
	}, []string{"url", "friendly_name", "type"})
)

func main() {
	var a app
	flag.StringVar(&a.apiKey, "api-key", "", "Uptime Robot API key")
	flag.StringVar(&a.address, "ip", "0.0.0.0", "IP on which the Prometheus server will be binded")
	flag.StringVar(&a.port, "p", "9705", "Port that will be used by the Prometheus server")
	flag.IntVar(&a.scrapeInterval, "inteval", 30, "Uptime robot API scrape interval, in seconds")
	flag.StringVar(&a.logLevel, "log-level", "info", "Log level")
	flag.Parse()

	a.logger = logger.New(a.logLevel)
	if a.apiKey == "" {
		a.apiKey = os.Getenv("UPTIMEROBOT_API_KEY")
		if a.apiKey == "" {
			a.logger.Fatal().Err(errors.New("missing Uptime Robot API key")).Msg("use -api-key or UPTIMEROBOT_API_KEY env variable")
		}
	}
	a.logger.Info().Msg("API key found")
	a.logger.Info().Msg("starting fetch routines")

	go a.fetchAccountDetails()
	go a.fetchMonitors()

	a.logger.Info().Msg("starting metrics server")
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "I'm alive! 8)")
	})
	http.ListenAndServe(a.address+":"+a.port, nil)
}

func (a app) fetchAccountDetails() {
	ticker := time.NewTicker(time.Duration(a.scrapeInterval) * time.Second)
	for {
		<-ticker.C
		a.logger.Info().Msg("fetching account details")
		data := url.Values{
			"api_key": {a.apiKey},
			"format":  {"json"},
		}

		resp, err := http.PostForm("https://api.uptimerobot.com/v2/getAccountDetails", data)
		if err != nil {
			a.logger.Error().Err(err).Msg("failed to fetch account details")
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			a.logger.Error().Err(err).Msg("cannot parse response body")
			continue
		}

		var account AccountDetails
		if err := json.Unmarshal(body, &account); err != nil {
			a.logger.Error().Err(err).Msg("cannot parse JSON")
			continue
		}

		a.logger.Debug().Msg("updating account details metrics")
		upMonitors.Set(float64(account.Account.UpMonitors))
		downMonitors.Set(float64(account.Account.DownMonitors))
		pausedMonitors.Set(float64(account.Account.PausedMonitors))

		accountDetails.WithLabelValues(account.Account.Firstname,
			account.Account.Email,
			strconv.Itoa(account.Account.MonitorLimit),
			strconv.Itoa(account.Account.MonitorInterval),
			strconv.Itoa(account.Account.UpMonitors),
			strconv.Itoa(account.Account.DownMonitors),
			strconv.Itoa(account.Account.PausedMonitors),
			strconv.Itoa(account.Account.PaymentPeriod))
	}
}

func (a app) fetchMonitors() {
	ticker := time.NewTicker(time.Duration(a.scrapeInterval) * time.Second)
	for {
		<-ticker.C
		logrus.Info("fetching monitors")
		data := url.Values{
			"api_key":              {a.apiKey},
			"format":               {"json"},
			"response_times":       {"1"},
			"response_times_limit": {"1"},
		}

		resp, err := http.PostForm("https://api.uptimerobot.com/v2/getMonitors", data)
		if err != nil {
			a.logger.Error().Err(err).Msg("failed to fetch monitors")
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			a.logger.Error().Err(err).Msg("cannot parse response body")
			continue
		}

		var monitors MonitorsData
		if err := json.Unmarshal(body, &monitors); err != nil {
			a.logger.Error().Err(err).Msg("cannot parse JSON")
			continue
		}

		for _, m := range monitors.Monitors {
			a.logger.Debug().Msg("updating monitors metrics")
			monitorsStatus.WithLabelValues(m.URL, m.FriendlyName, strconv.Itoa(m.Interval)).Set(float64(m.Status))
			responseTime.WithLabelValues(m.URL, m.FriendlyName, strconv.Itoa(m.Type)).Set(float64(m.ResponseTimes[0].Value))
		}
	}
}
