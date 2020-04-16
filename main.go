package main

import (
	"flag"
	"log"
	"net/http"
	"os"
)

// var addr = flag.String("addr", "127.0.0.1:8080", "http service address")

func servePublic(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.FileServer(http.Dir("./public"))
}

func main() {
	flag.Parse()
	hub := newHub()
	go hub.run()
	http.HandleFunc("/", servePublic)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on port %s", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	// if err := http.ListenAndServe(":"+port, nil); err != nil {
	// 	log.Fatal("ListenAndServe: ", err)
	// }
}
