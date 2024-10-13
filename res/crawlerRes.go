package res

import (
	"github.com/Kumengda/easyChromedp/template"
)

type BaseUrl struct {
	Url string `json:"url"`
}
type SameOriginUrl struct {
	BaseUrl
}

type ExternalLink struct {
	BaseUrl
}

type ExternalStaticFileLink struct {
	BaseUrl
}

type DirResult struct {
	Target                 string                   `json:"target"`
	SameOriginUrl          []SameOriginUrl          `json:"same_originUrl"`
	ExternalLink           []ExternalLink           `json:"external_link"`
	ExternalStaticFileLink []ExternalStaticFileLink `json:"external_static_file_link"`
	SameOriginForm         []template.JsRes         `json:"same_origin_form"`
	ExternalForm           []template.JsRes         `json:"external_form"`
}
