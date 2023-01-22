package web

import (
	"github.com/lpar/gzipped/v2"
	"github.com/mattermost/gziphandler"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
	"github.com/mattermost/mattermost-server/v6/utils"
	"github.com/mattermost/mattermost-server/v6/utils/fileutils"
	"net/http"
	"path"
)

func (w *Web) InitStatic() {
	if *w.srv.Config().ServiceSettings.WebserverMode != "disabled" {
		if err := utils.UpdateAssetsSubpathFromConfig(w.srv.Config()); err != nil {
			mlog.Error("Failed to update assets subpath from config", mlog.Err(err))
		}

		staticDir, _ := fileutils.FindDir(model.ClientDir)
		mlog.Debug("Using client directory", mlog.String("clientDir", staticDir))

		subpath, _ := utils.GetSubpathFromConfig(w.srv.Config())

		staticHandler := staticFilesHandler(http.StripPrefix(path.Join(subpath, "static"), gzipped.FileServer(gzipped.Dir(staticDir))))
		pluginHandler := staticFilesHandler(http.StripPrefix(path.Join(subpath, "static", "plugins"), gzipped.FileServer(gzipped.Dir(*w.srv.Config().PluginSettings.ClientDirectory))))

		if *w.srv.Config().ServiceSettings.WebserverMode == "gzip" {
			staticHandler = gziphandler.GzipHandler(staticHandler)
			pluginHandler = gziphandler.GzipHandler(pluginHandler)
		}

		w.MainRouter.PathPrefix("/static/plugins/").Handler(pluginHandler)
		w.MainRouter.PathPrefix("/static/").Handler(staticHandler)
		w.MainRouter.Handle("/robots.txt", http.HandlerFunc(robotsHandler))
		w.MainRouter.Handle("/unsupported_browser.js", http.HandlerFunc(unsupportedBrowserScriptHandler))
		w.MainRouter.Handle("/{anything:.*}", w.NewStaticHandler(root)).Methods("GET")

		// When a subpath is defined, it's necessary to handle redirects without a
		// trailing slash. We don't want to use StrictSlash on the w.MainRouter and affect
		// all routes, just /subpath -> /subpath/.
		w.MainRouter.HandleFunc("", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path += "/"
			http.Redirect(w, r, r.URL.String(), http.StatusFound)
		}))
	}
}
