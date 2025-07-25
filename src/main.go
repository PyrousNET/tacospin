package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

var (
	spins      float64
	spinning   bool
	rpm        float64
	mutex      sync.Mutex
	stopSignal chan bool
)

const (
	windmillDiameter = 1.0 // in meters
)

type EntriesItem struct {
	Humidity       string `json:"humidity"`
	Luminosity     string `json:"luminosity"`
	Name           string `json:"name"`
	Pressure       string `json:"pressure"`
	Rain_1h        string `json:"rain_1h"`
	Rain_24h       string `json:"rain_24h"`
	Rain_mn        string `json:"rain_mn"`
	Temp           string `json:"temp"`
	Time           string `json:"time"`
	Wind_direction string `json:"wind_direction"`
	Wind_gust      string `json:"wind_gust"`
	Wind_speed     string `json:"wind_speed"`
}

type Weather struct {
	Command string        `json:"command"`
	Entries []EntriesItem `json:"entries"`
	Found   float64       `json:"found"`
	Result  string        `json:"result"`
	What    string        `json:"what"`
}

func getWindSpeed(url string) (float64, error) {
	if url == "" {
		// return random wind speed if no URL is provided
		return float64(5 + rand.Intn(15)), nil // Random wind speed between 5 and 20 m/s
	}
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data Weather
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	var wind string
	for _, entry := range data.Entries {
		wind = entry.Wind_speed
	}
	var windSpeed float64
	if wind != "" {
		windSpeed, err = strconv.ParseFloat(wind, 64)
	}
	return windSpeed, nil
}

func spinCalculator(windSpeed float64) float64 {
	const TSR = 8.0 // Assume fast-spinning unloaded rotor
	return (TSR * windSpeed) / (math.Pi * windmillDiameter) * 60.0
}

func startSpinning() {
	url := ""
	apiKey := os.Getenv("OPENWEATHER_API_KEY")
	if apiKey != "" {
		url = fmt.Sprintf("https://api.aprs.fi/api/get?name=GW2066&what=wx&apikey=%s", apiKey)
	}
	mutex.Lock()
	if spinning {
		mutex.Unlock()
		return
	}
	spinning = true
	stopSignal = make(chan bool)
	mutex.Unlock()

	go func() {
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()
		spinTicker := time.NewTicker(1 * time.Second)
		defer spinTicker.Stop()
		for {
			select {
			case <-stopSignal:
				return
			case <-ticker.C:
				windSpeed, err := getWindSpeed(url)
				if err != nil {
					log.Println("Error getting wind speed:", err)
					continue
				}
				rpmVal := spinCalculator(windSpeed)
				mutex.Lock()
				rpm = rpmVal
				mutex.Unlock()
			case <-spinTicker.C:
				mutex.Lock()
				spins += rpm / 60.0
				mutex.Unlock()
			}
		}
	}()
	windSpeed, err := getWindSpeed(url)
	if err != nil {
		log.Println("Error getting wind speed:", err)
		return
	}
	rpmVal := spinCalculator(windSpeed)
	mutex.Lock()
	spins += rpmVal / 60.0
	rpm = rpmVal
	mutex.Unlock()
}

func stopSpinning() {
	mutex.Lock()
	if spinning {
		spinning = false
		close(stopSignal)
	}
	mutex.Unlock()
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}

func main() {
	_ = godotenv.Load()
	r := chi.NewRouter()

	r.Get("/", serveIndex)

	r.Post("/start", func(w http.ResponseWriter, r *http.Request) {
		startSpinning()
		w.Write([]byte("Spinning started"))
	})

	r.Post("/stop", func(w http.ResponseWriter, r *http.Request) {
		stopSpinning()
		w.Write([]byte("Spinning stopped"))
	})

	r.Get("/spins", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		s := spins
		mutex.Unlock()
		w.Write([]byte(fmt.Sprintf("Total taco spins: %.2f", s)))
	})

	r.Get("/rpm", func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		rot := rpm
		mutex.Unlock()
		w.Write([]byte(fmt.Sprintf("%.2f", rot)))
	})

	fs := http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))
	r.Handle("/static/*", fs)

	log.Println("Taco spin service started on :8080")
	http.ListenAndServe(":8080", r)
}
