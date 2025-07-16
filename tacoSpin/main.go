package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

var (
	spins      float64
	spinning   bool
	mutex      sync.Mutex
	stopSignal chan bool
)

const (
	windmillDiameter = 1.0 // in meters
)

type Clouds struct {
	All float64 `json:"all"`
}

type Coord struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Main struct {
	Feels_like float64 `json:"feels_like"`
	Grnd_level float64 `json:"grnd_level"`
	Humidity float64 `json:"humidity"`
	Pressure float64 `json:"pressure"`
	Sea_level float64 `json:"sea_level"`
	Temp float64 `json:"temp"`
	Temp_max float64 `json:"temp_max"`
	Temp_min float64 `json:"temp_min"`
}

type Sys struct {
	Country string `json:"country"`
	Id float64 `json:"id"`
	Sunrise float64 `json:"sunrise"`
	Sunset float64 `json:"sunset"`
	Type float64 `json:"type"`
}

type WeatherItem struct {
	Description string `json:"description"`
	Icon string `json:"icon"`
	Id float64 `json:"id"`
	Main string `json:"main"`
}

type Wind struct {
	Deg float64 `json:"deg"`
	Gust float64 `json:"gust"`
	Speed float64 `json:"speed"`
}

type Weather struct {
	Base string `json:"base"`
	Clouds Clouds `json:"clouds"`
	Cod float64 `json:"cod"`
	Coord Coord `json:"coord"`
	Dt float64 `json:"dt"`
	Id float64 `json:"id"`
	Main Main `json:"main"`
	Name string `json:"name"`
	Sys Sys `json:"sys"`
	Timezone float64 `json:"timezone"`
	Visibility float64 `json:"visibility"`
	Weather []WeatherItem `json:"weather"`
	Wind Wind `json:"wind"`
}


type windResponse struct {
	Current struct {
		WindSpeed float64 `json:"wind_speed"`
	} `json:"current"`
}

func getWindSpeed() (float64, error) {
	apiKey := os.Getenv("OPENWEATHER_API_KEY")
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=Chicago&appid=%s&units=metric", apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data Weather
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	wind := data.Wind
	return wind.Speed, nil
}

func spinCalculator(windSpeed float64) float64 {
	// assume RPM = wind speed (m/s) / (pi * D) * 60
	return windSpeed / (math.Pi * windmillDiameter) * 60.0
}

func startSpinning() {
	mutex.Lock()
	if spinning {
		mutex.Unlock()
		return
	}
	spinning = true
	stopSignal = make(chan bool)
	mutex.Unlock()
    windSpeed, err := getWindSpeed()
    if err != nil {
        log.Println("Error getting wind speed:", err)
        return
    }

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopSignal:
				return
			case <-ticker.C:
				rpm := spinCalculator(windSpeed)
				mutex.Lock()
				spins += rpm / 60.0 // spins per second
				mutex.Unlock()
			}
		}
	}()
}

func stopSpinning() {
	mutex.Lock()
	if spinning {
		spinning = false
		close(stopSignal)
	}
	mutex.Unlock()
}

func main() {
	_ = godotenv.Load()
	r := chi.NewRouter()

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

	log.Println("Taco spin service started on :8080")
	http.ListenAndServe(":8080", r)
}

