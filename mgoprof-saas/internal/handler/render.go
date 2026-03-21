package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// renderAdminPage renders an admin UI page. When embedded HTML templates are
// not yet available (early development) it falls back to a minimal HTML page
// that serialises the page data as formatted JSON — enough to verify the
// handler logic without requiring a frontend build step.
//
// Replace the body of this function with a real template engine
// (html/template + fs.FS embed) once the admin frontend is ready.
func renderAdminPage(w http.ResponseWriter, page string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		pretty = []byte(fmt.Sprintf(`{"error": %q}`, err.Error()))
	}

	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <title>Admin — %s</title>
  <style>
    body { font-family: monospace; background: #1e1e1e; color: #d4d4d4; padding: 2rem; }
    h1   { color: #9cdcfe; }
    pre  { background: #252526; padding: 1rem; border-radius: 6px; overflow-x: auto; }
  </style>
</head>
<body>
  <h1>Admin panel: %s</h1>
  <pre>%s</pre>
</body>
</html>`, page, page, string(pretty))
}
