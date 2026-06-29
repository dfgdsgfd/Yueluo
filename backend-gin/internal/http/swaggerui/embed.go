package swaggerui

import (
	"embed"
	"path"
	"sync"
)

const scalarVersion = "1.62.0"

//go:embed swagger-ui.css swagger-ui-bundle.js swagger-ui-standalone-preset.js scalar-api-reference.css scalar-api-reference.js
var assets embed.FS

var (
	once           sync.Once
	cssData        []byte
	jsData         []byte
	presetData     []byte
	scalarCSSData  []byte
	scalarJSData   []byte
	cssError       error
	jsError        error
	presetError    error
	scalarCSSError error
	scalarJSError  error
)

func load() {
	cssData, cssError = assets.ReadFile("swagger-ui.css")
	jsData, jsError = assets.ReadFile("swagger-ui-bundle.js")
	presetData, presetError = assets.ReadFile("swagger-ui-standalone-preset.js")
	scalarCSSData, scalarCSSError = assets.ReadFile("scalar-api-reference.css")
	scalarJSData, scalarJSError = assets.ReadFile("scalar-api-reference.js")
}

func CSS() ([]byte, error) {
	once.Do(load)
	return cssData, cssError
}

func JS() ([]byte, error) {
	once.Do(load)
	return jsData, jsError
}

func StandalonePreset() ([]byte, error) {
	once.Do(load)
	return presetData, presetError
}

func ScalarCSS() ([]byte, error) {
	once.Do(load)
	return scalarCSSData, scalarCSSError
}

func ScalarJS() ([]byte, error) {
	once.Do(load)
	return scalarJSData, scalarJSError
}

func ScalarUIQuery() string {
	return "scalar-" + scalarVersion
}

func SwaggerUIPath() string {
	return "/api/swagger-ui"
}

func CSSPath() string {
	return path.Join(SwaggerUIPath(), "swagger-ui.css")
}

func JSPath() string {
	return path.Join(SwaggerUIPath(), "swagger-ui-bundle.js")
}

func StandalonePresetPath() string {
	return path.Join(SwaggerUIPath(), "swagger-ui-standalone-preset.js")
}

func ScalarCSSPath() string {
	return CSSPath() + "?ui=" + ScalarUIQuery()
}

func ScalarJSPath() string {
	return JSPath() + "?ui=" + ScalarUIQuery()
}
