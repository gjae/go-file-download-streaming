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

	"github.com/gjae/easydownloadmanager/server"
	"github.com/gorilla/mux"
)

//go:embed templates
var templateFs embed.FS

//go:embed downloads
var downloadFolder embed.FS

// RegisterHandlers sets up the routes and their corresponding handlers for the MuxServer.
// It registers a handler for the root path ("/") to serve the index.html template,
// and a handler for "/download/{filename}" to serve downloadable files from the embedded 'downloads' folder.
func RegisterHandlers(muxServer *server.MuxServer) {
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

		w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

		io.Copy(w, file)
	})
}

func main() {
	var hostFlag string
	var portFlag string
	ctx, cancel := context.WithCancel(context.Background())

	hostFlag = "0.0.0.0"
	portFlag = "8081"
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
