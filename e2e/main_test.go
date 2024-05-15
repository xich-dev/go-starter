//go:build !ut
// +build !ut

package e2e

import (
	"log"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/xich-dev/go-starter/pkg/apigen"
	"github.com/xich-dev/go-starter/pkg/apps/server"
	"github.com/xich-dev/go-starter/wire"
)

func newHttpExpect(handler http.Handler, t *testing.T) *httpexpect.Expect {
	return httpexpect.WithConfig(httpexpect.Config{
		Client: &http.Client{
			Transport: httpexpect.NewBinder(handler),
			Jar:       httpexpect.NewCookieJar(),
		},
		Reporter: httpexpect.NewRequireReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		},
	})
}

var (
	apiServer *server.Server
	token     string
	mu        sync.Mutex
)

var (
	globalPhone    = "18688338517"
	globalUsername = "sage"
	globalPassword = "1234"
)

func TestMain(m *testing.M) {
	server, err := wire.InitializeServer()
	if err != nil {
		log.Fatal(err)
	}
	apiServer = server

	os.Exit(m.Run())
}

func getTestEngine(t *testing.T) *httpexpect.Expect {
	endpoint := os.Getenv("TEST_ENDPOINT")
	if endpoint != "" {
		return httpexpect.WithConfig(httpexpect.Config{
			BaseURL:  endpoint,
			Reporter: httpexpect.NewRequireReporter(t),
			Printers: []httpexpect.Printer{
				httpexpect.NewDebugPrinter(t, true),
			},
		})

	}
	return newHttpExpect(adaptor.FiberApp(apiServer.GetApp()), t)
}

type AutenticatedTestEngine struct {
	*httpexpect.Expect
	authInfo apigen.AuthInfo
}

func getAuthenticatedTestEngine(t *testing.T) *AutenticatedTestEngine {
	te := getTestEngine(t)
	mu.Lock()
	tmpToken := token
	mu.Unlock()

	if tmpToken == "" {
		registerAccount(t, globalPhone, globalUsername, globalPassword)
	}

	authInfo := loginAccount(t, globalPhone, globalUsername, globalPassword)

	return &AutenticatedTestEngine{
		Expect: te.Builder(func(req *httpexpect.Request) {
			req.WithHeader("Authorization", "Bearer "+token)
		}),
		authInfo: authInfo,
	}
}
