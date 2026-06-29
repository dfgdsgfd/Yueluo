package routes

import (
	"embed"
	"encoding/json"
)

//go:embed route-matrix.json
var matrixFS embed.FS

type Matrix struct {
	Summary    Summary          `json:"summary"`
	Routes     []Route          `json:"routes"`
	WebSockets []WebSocketEntry `json:"webSockets"`
}

type Summary struct {
	MountedModules int            `json:"mountedModules"`
	ModuleRoutes   int            `json:"moduleRoutes"`
	InlineRoutes   int            `json:"inlineRoutes"`
	TotalAPIRoutes int            `json:"totalApiRoutes"`
	MethodCounts   map[string]int `json:"methodCounts"`
	ModuleCounts   map[string]int `json:"moduleCounts"`
}

type Route struct {
	Method     string   `json:"method"`
	Path       string   `json:"path"`
	SourceFile string   `json:"sourceFile"`
	Line       int      `json:"line"`
	Middleware []string `json:"middleware"`
	Auth       string   `json:"auth"`
	Type       string   `json:"type"`
	Status     string   `json:"status"`
	Notes      string   `json:"notes"`
}

type WebSocketEntry struct {
	Path       string `json:"path"`
	SourceFile string `json:"sourceFile"`
	Line       int    `json:"line"`
	Auth       string `json:"auth"`
	Status     string `json:"status"`
	Notes      string `json:"notes"`
}

func LoadMatrix() (Matrix, error) {
	data, err := matrixFS.ReadFile("route-matrix.json")
	if err != nil {
		return Matrix{}, err
	}
	var matrix Matrix
	if err := json.Unmarshal(data, &matrix); err != nil {
		return Matrix{}, err
	}
	return matrix, nil
}
