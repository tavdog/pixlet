package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"tidbyt.dev/pixlet/runtime"
	"tidbyt.dev/pixlet/server/loader"
)

func init() {
	ApiCmd.Flags().StringVarP(&host, "host", "i", "127.0.0.1", "Host interface for serving rendered images")
	ApiCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port for serving rendered images")
	ApiCmd.Flags().BoolVarP(&renderGif, "gif", "", false, "Generate GIF instead of WebP")
	ApiCmd.Flags().BoolVarP(&silenceOutput, "silent", "", false, "Silence print statements when rendering app")
}

var ApiCmd = &cobra.Command{
	Use:   "api",
	Short: "Run a Pixlet API server",
	Args:  cobra.MinimumNArgs(0),
	RunE:  api,
	Long: `Start an HTTP server that runs a Pixlet app in response to API requests.
	`,
}

type renderRequest struct {
	Path    string            `json:"path"`
	Config  map[string]string `json:"config"`
	Width   int               `json:"width"`
	Height  int               `json:"height"`
	Magnify int               `json:"magnify"`
}

func validatePath(path string) bool {
	return !strings.Contains(path, "..")
}

// Example request
//
//	{
//	   "path": "/workspaces/pixlet/examples/clock",
//	   "config": {
//	       "timezone": "America/New_York"
//	   }
//	}
func renderHandler(w http.ResponseWriter, req *http.Request) {
	var r renderRequest

	if err := json.NewDecoder(req.Body).Decode(&r); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode render request: %v", err), http.StatusBadRequest)
		return
	}

	if !validatePath(r.Path) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	buf, err := loader.RenderApplet(r.Path, r.Config, r.Width, r.Height, r.Magnify, maxDuration, timeout, renderGif, silenceOutput)
	if err != nil {
		http.Error(w, fmt.Sprintf("error rendering: %v", err), http.StatusInternalServerError)
		return
	}

	if renderGif {
		w.Header().Set("Content-Type", "image/gif")
	} else {
		w.Header().Set("Content-Type", "image/webp")
	}
	w.Write(buf)
}

func api(cmd *cobra.Command, args []string) error {
	cache := runtime.NewInMemoryCache()
	runtime.InitHTTP(cache)
	runtime.InitCache(cache)

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("listening at http://%s\n", addr)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/render", renderHandler)
	return http.ListenAndServe(addr, mux)
}
