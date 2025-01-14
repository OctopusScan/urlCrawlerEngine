package runtime

import (
	"github.com/B9O2/Inspector"
	"github.com/B9O2/Inspector/decorators"
	"github.com/B9O2/Inspector/types"
)

var MainInsp = inspect.NewInspector("crawler", 9999)

var (
	FileName = types.NewType("file_name", func(i interface{}) string {
		return "  " + i.(string) + "  :  "
	}, decorators.Magenta)
)
