//go:build liteconpty && amd64

package claudetui

import (
	_ "embed"
	"strings"
)

// The ConPTY magpie-lite ships (Microsoft.Windows.Console.ConPTY, x64, MIT),
// put in conpty-bundle by build/lite/package-windows.sh for its build.
var (
	//go:embed conpty-bundle/version
	bundleVersion string
	//go:embed conpty-bundle/conpty.dll
	bundleDLL []byte
	//go:embed conpty-bundle/OpenConsole.exe
	bundleHost []byte
	//go:embed conpty-bundle/LICENSE.txt
	bundleLicense []byte
)

func init() {
	bundle.version = strings.TrimSpace(bundleVersion)
	bundle.dll, bundle.host, bundle.license = bundleDLL, bundleHost, bundleLicense
}
