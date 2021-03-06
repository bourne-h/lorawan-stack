// Copyright © 2019 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

import (
	"context"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/labstack/echo"
	echomiddleware "github.com/labstack/echo/middleware"
	"go.thethings.network/lorawan-stack/pkg/config"
	"go.thethings.network/lorawan-stack/pkg/errors"
	"go.thethings.network/lorawan-stack/pkg/log"
	"go.thethings.network/lorawan-stack/pkg/random"
	"go.thethings.network/lorawan-stack/pkg/web/cookie"
	"go.thethings.network/lorawan-stack/pkg/web/middleware"
)

// Registerer allows components to register their services to the web server.
type Registerer interface {
	RegisterRoutes(s *Server)
}

// Server is the server.
type Server struct {
	*rootGroup
	config config.HTTP
	server *echo.Echo
}

type rootGroup struct {
	*echo.Group
}

// New builds a new server.
func New(ctx context.Context, config config.HTTP) (*Server, error) {
	logger := log.FromContext(ctx).WithField("namespace", "web")

	hashKey, blockKey := config.Cookie.HashKey, config.Cookie.BlockKey

	if len(hashKey) == 0 || isZeros(hashKey) {
		hashKey = random.Bytes(64)
		logger.WithField("hash_key", hashKey).Warn("No cookie hash key configured, generated a random one")
	}

	if len(hashKey) != 32 && len(hashKey) != 64 {
		return nil, errors.New("Expected cookie hash key to be 32 or 64 bytes long")
	}

	if len(blockKey) == 0 || isZeros(blockKey) {
		blockKey = random.Bytes(32)
		logger.WithField("block_key", blockKey).Warn("No cookie block key configured, generated a random one")
	}

	if len(blockKey) != 32 {
		return nil, errors.New("Expected cookie block key to be 32 bytes long")
	}

	server := echo.New()

	server.Logger = &noopLogger{}
	server.HTTPErrorHandler = ErrorHandler

	server.Use(
		middleware.ID(""),
		echomiddleware.BodyLimit("16M"),
		echomiddleware.Secure(),
		echomiddleware.Recover(),
		cookie.Cookies(blockKey, hashKey),
	)

	s := &Server{
		rootGroup: &rootGroup{
			Group: server.Group(
				"",
				middleware.Log(logger),
				middleware.Normalize(middleware.RedirectPermanent),
			),
		},
		config: config,
		server: server,
	}

	var staticDir http.Dir
	for _, path := range config.Static.SearchPath {
		if s, err := os.Stat(path); err == nil && s.IsDir() {
			staticDir = http.Dir(path)
			break
		}
	}
	if staticDir != "" {
		logger.WithFields(log.Fields("path", staticDir, "mount", config.Static.Mount)).Debug("Serving static assets")
		s.Static(config.Static.Mount, staticDir, middleware.Immutable)
	} else {
		logger.WithField("search_path", config.Static.SearchPath).Warn("No static assets found in any search path")
	}

	return s, nil
}

func isZeros(buf []byte) bool {
	for _, b := range buf {
		if b != 0x00 {
			return false
		}
	}

	return true
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.server.ServeHTTP(w, r)
}

// Group creates a sub group.
func (s *Server) Group(prefix string, middleware ...echo.MiddlewareFunc) *echo.Group {
	t := strings.TrimSuffix(prefix, "/")
	return s.rootGroup.Group.Group(t, middleware...)
}

// RootGroup creates a new Echo router group with prefix and optional group-level middleware on the root Server.
func (s *Server) RootGroup(prefix string, middleware ...echo.MiddlewareFunc) *echo.Group {
	t := strings.TrimSuffix(prefix, "/")
	return s.server.Group(t, middleware...)
}

// Static adds the http.FileSystem under the defined prefix.
func (s *Server) Static(prefix string, fs http.FileSystem, middleware ...echo.MiddlewareFunc) {
	t := strings.TrimSuffix(prefix, "/")
	path := path.Join(t, "*")
	fileServer := http.StripPrefix(t, http.FileServer(fs))
	handler := func(c echo.Context) error {
		fileServer.ServeHTTP(c.Response().Writer, c.Request())
		return nil
	}
	s.GET(path, handler, middleware...)
	s.HEAD(path, handler, middleware...)
}

// Routes returns the defined routes.
func (s *Server) Routes() []*echo.Route {
	return s.server.Routes()
}
