/*
 * Author: Ben Payne [trixtur@gmail.com]
 * Date: 4-24-2023
 *
 * CoAuthor: Josh Payne []
 * Date: 4-24-2023
 *
 * Description: A simple taco spinning server with web hooks.
 */
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Counter struct {
	count     uint64
	rollovers uint64
}

type Server struct {
	Counter  *Counter
	Result   *Result
	upgrader websocket.Upgrader
}

func NewServer(counter *Counter) *Server {
	return &Server{
		Counter: counter,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (c *Counter) Increment() {
	c.count++
}

func (c *Counter) Reset() {
	c.count = 0
	c.rollovers = 0
}

func (c *Counter) GetRollovers() uint64 {
	return c.rollovers
}

type Result struct {
	Start      int64  `json:"start"`
	End        int64  `json:"end"`
	TotalCount uint64 `json:"total_count"`
}

func (r *Result) Restart() {
	time := time.Now()

	r.Start = time.Unix()
	r.End = 0
}

func (r *Result) Finish() {
	time := time.Now()

	r.End = time.Unix()
}

func (r *Result) ComputeTotal(counter Counter) {
	// Compute the total count as the sum of the count and the maximum value of the counter multiplied by the number of rollovers
	totalCount := counter.count + (^uint64(0) * counter.rollovers)

	r.TotalCount = totalCount
}

type TacoPageData struct {
	Class   string
	Message string
}

func main() {
	const Reset = "\033[0m"
	const Red = "\033[31m"
	const Green = "\033[32m"
	const Yellow = "\033[33m"
	const Blue = "\033[34m"
	const Purple = "\033[35m"
	const Cyan = "\033[36m"
	const Gray = "\033[37m"
	const White = "\033[97m"

	rand.Seed(time.Now().UnixNano())

	counter := &Counter{}
	result := &Result{}

	server := NewServer(counter)
	server.Result = result

	tmpl := template.Must(template.ParseFiles("public/taco.html"))

	// Start the daemon in a goroutine
	go func() {
		for {
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
			if result.End == 0 && result.Start != 0 {
				counter.Increment()
			}
		}
	}()

	// HTTP endpoint for setting the start time
	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		counter.Reset()

		result.Restart()
		fmt.Print(Yellow + "The mighty taco has started to spin at " + Reset)
		fmt.Printf(Gray+"%d\n"+Reset, result.Start)

		fmt.Fprintf(w, "Start time set to: %d\n", result.Start)
	})

	// HTTP endpoint for setting the end time and retrieving the count difference
	http.HandleFunc("/end", func(w http.ResponseWriter, r *http.Request) {
		if result.Start == 0 {
			http.Error(w, "the mighty taco has not yet started to spin", http.StatusBadRequest)
			return
		}

		if result.End != 0 {
			http.Error(w, "the mighty taco has alredy made its decision", http.StatusBadRequest)
			return
		}

		result.Finish()
		fmt.Print(Yellow + "The mighty taco has completed its rotations at " + Reset)
		fmt.Printf(Gray+"%d\n"+Reset, result.End)

		result.ComputeTotal(*server.Counter)
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBytes)
	})

	http.HandleFunc("/ws", server.handleWebSocket)

	http.HandleFunc("/taco.jpg", func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile("public/taco.jpg")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(buf)
	})

	http.HandleFunc("/taco.css", func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile("public/css/taco.css")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/css")
		w.Write(buf)
	})

	http.HandleFunc("/taco.js", func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile("public/js/taco.js")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/javascript")
		w.Write(buf)
	})

	// HTTP endpoint for retrieving the current start, end and total
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		result.ComputeTotal(*counter)
		jsonBytes, err := json.Marshal(result)
		data := TacoPageData{Class: "", Message: ""}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if result.Start == 0 {
			data.Message = "The mighty taco has not yet started to spin."
			tmpl.Execute(w, data)
			//http.Error(w, "the mighty taco has not yet started to spin", http.StatusBadRequest)
			return
		}

		data.Class = "spin"
		data.Message = fmt.Sprintf("The mighty taco spins have been observed at %d rotations", result.TotalCount)
		tmpl.Execute(w, data)

		fmt.Print(Green + "The mighty taco spins have been observed at " + Reset)
		fmt.Printf(Gray+"%d "+Reset, result.TotalCount)
		fmt.Println(Green + "rotations." + Reset)
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonBytes)
	})

	fmt.Println(Green + "Server listening on port 8080" + Reset)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("error upgrading to WebSocket:", err)
		return
	}

	go s.sendCounterIncrement(conn)
}

func (s *Server) sendCounterIncrement(conn *websocket.Conn) {
	for {
		time.Sleep(3 * time.Second)
		if s.Result.End == 0 && s.Result.Start != 0 {
			s.Result.ComputeTotal(*s.Counter)
			result := map[string]uint{
				"start":       uint(s.Result.Start),
				"total_count": uint(s.Counter.count + (^uint64(0) * s.Counter.rollovers)),
			}

			jsonResult, err := json.Marshal(result)
			if err != nil {
				log.Println("error marshaling result:", err)
				continue
			}

			err = conn.WriteMessage(websocket.TextMessage, jsonResult)
			if err != nil {
				log.Println("error sending WebSocket message:", err)
				break
			}
		}
	}
}
