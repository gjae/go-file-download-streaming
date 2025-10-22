package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/gjae/easydownloadmanager/server"
	"github.com/gorilla/mux"
)

//go:embed templates
var templateFs embed.FS

//go:embed downloads
var downloadFolder embed.FS

// WriteHeader sets the appropriate HTTP headers for file downloads.
// It configures Content-Disposition for attachment, Content-Type for binary stream,
// and Content-Length with the file size.
func WriteHeader(w http.ResponseWriter, size int64, filename string) {
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
}

// RegisterHandlers sets up the routes and their corresponding handlers for the MuxServer.
// It registers handlers for:
// - Root path ("/"): serves the index.html template with server info
// - "/download/limited/{filename}": serves files with throttled speed (512KB/s) and progress logging
// - "/download/{filename}": serves files at maximum speed using io.Copy
func RegisterHandlers(muxServer *server.MuxServer) {
	// Root handler serves the main page with download links
	muxServer.HandlerFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFS(templateFs, "templates/index.html")
		if err != nil {
			log.Println(err)
			return
		}
		tmplContext := make(map[string]string)
		tmplContext["host"] = muxServer.Host
		tmplContext["Port"] = muxServer.Port
		tmplContext["filename"] = "nature-3082832_1280.jpg"

		tmpl.Execute(w, tmplContext)
	})

	// Limited download handler demonstrates manual streaming with speed throttling
	// Uses a 512KB buffer and 1-second delays between chunks to achieve ~512KB/s speed
	// Includes progress logging to track download status in real-time
	muxServer.HandlerFunc("/download/limited/{filename}", func(w http.ResponseWriter, r *http.Request) {
		filename := mux.Vars(r)["filename"]
		buffer := make([]byte, 200*1024) // 512KB buffer for controlled chunk size
		filePath := "downloads" + "/" + filename
		var totalDownloaded int64 // Tracks total bytes sent for progress reporting

		log.Printf("Buscando: %s", filePath)
		file, err := downloadFolder.Open(filePath)
		if err != nil {
			log.Println()
			w.Write([]byte("404: File not found"))
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			log.Printf("ERROR: file not found: %v", err)
			return
		}
		WriteHeader(w, info.Size(), filename)
		flusher, hasFlusher := w.(http.Flusher)

		// Manual streaming loop with progress tracking and speed control
		for {
			n, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				log.Printf("Error reading file: %v", err)
				return
			}
			if n == 0 {
				break
			}

			totalDownloaded += int64(n)
			// Log progress with percentage completion
			log.Printf("Downloaded total: %d bytes (%.1f%%) \n...", totalDownloaded, float64(totalDownloaded)/float64(info.Size())*100)
			w.Write(buffer[:n])
			if hasFlusher {
				flusher.Flush() // Immediate send to client
			}
			time.Sleep(1 * time.Second) // Throttle to ~512KB/s
		}
	})

	// Normal download handler uses io.Copy for maximum performance
	// Serves files at the highest possible speed without throttling
	muxServer.HandlerFunc("/download/{filename}", func(w http.ResponseWriter, r *http.Request) {
		filename := mux.Vars(r)["filename"]

		// Ruta exacta para embed
		filePath := "downloads/" + filename

		log.Printf("Buscando: %s", filePath)

		file, err := downloadFolder.Open(filePath)
		if err != nil {
			log.Printf("ERROR: No se pudo abrir %s: %v", filePath, err)
			http.Error(w, "Archivo no encontrado: "+filePath, http.StatusNotFound)
			return
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			log.Printf("ERROR en stat: %v", err)
			http.Error(w, "Error interno", http.StatusInternalServerError)
			return
		}

		log.Printf("ÉXITO: Archivo %s encontrado (%d bytes)", filename, info.Size())
		WriteHeader(w, info.Size(), filename)
		io.Copy(w, file) // Maximum speed transfer
	})
}

func main() {
	var hostFlag string
	var portFlag string
	ctx, cancel := context.WithCancel(context.Background())

	hostFlag = "0.0.0.0"
	portFlag = "8080"
	flag.StringVar(&hostFlag, "host", hostFlag, "Indica el host del servidor")
	flag.StringVar(&portFlag, "port", portFlag, "Indica el puerto del servidor")
	flag.Parse()

	fmt.Println("Hello world")
	muxServer := server.NewMuxServer(portFlag, hostFlag, nil)

	RegisterHandlers(muxServer)

	muxServer.Run(func() {
		log.Println("Shutting down server")
	}, func() {
		log.Println("Shuted down")
		cancel()
	}, ctx)
}
