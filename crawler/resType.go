package crawler

import (
	"github.com/Kumengda/easyChromedp/template"
	"net/http"
)

type Res struct {
	JsRes       template.JsRes
	RawResponse *http.Response
}
