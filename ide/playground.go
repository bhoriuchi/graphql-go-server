package ide

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

// PlaygroundVersion the default version to use
var PlaygroundVersion = "1.7.28"

type PlaygroundOptions struct {
	Version              string
	SSL                  bool
	Endpoint             string
	SubscriptionEndpoint string
}

func NewDefaultPlaygroundOptions() *PlaygroundOptions {
	return &PlaygroundOptions{
		Version: PlaygroundVersion,
	}
}

func NewDefaultSSLPlaygroundOptions() *PlaygroundOptions {
	return &PlaygroundOptions{
		Version: PlaygroundVersion,
		SSL:     true,
	}
}

type playgroundData struct {
	PlaygroundVersion    string
	Endpoint             string
	SubscriptionEndpoint string
	SetTitle             bool
}

// renderPlayground renders the Playground GUI
func RenderPlayground(config *PlaygroundOptions, w http.ResponseWriter, r *http.Request) {
	t := template.New("Playground")
	t, err := t.Parse(graphcoolPlaygroundTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	endpoint := r.URL.Path
	if config.Endpoint != "" {
		endpoint = config.Endpoint
	}

	wsScheme := "ws:"
	if config.SSL {
		wsScheme = "wss:"
	}

	subscriptionEndpoint := fmt.Sprintf("%s//%v%s", wsScheme, r.Host, r.URL.Path)
	if config.SubscriptionEndpoint != "" {
		subscriptionEndpoint = config.SubscriptionEndpoint
	}

	version := ""
	if config.Version != "" {
		version = fmt.Sprintf("@%s", strings.TrimLeft(config.Version, "@"))
	}

	d := playgroundData{
		PlaygroundVersion:    version,
		Endpoint:             endpoint,
		SubscriptionEndpoint: subscriptionEndpoint,
		SetTitle:             true,
	}
	err = t.ExecuteTemplate(w, "index", d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const graphcoolPlaygroundTemplate = `
{{ define "index" }}
<!DOCTYPE html>
<html>

<head>
  <meta charset=utf-8/>
  <meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
  <title>GraphQL Playground</title>
  <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/graphql-playground-react{{ .PlaygroundVersion }}/build/static/css/index.css" />
  <link rel="shortcut icon" href="//cdn.jsdelivr.net/npm/graphql-playground-react{{ .PlaygroundVersion }}/build/favicon.png" />
  <script src="//cdn.jsdelivr.net/npm/graphql-playground-react{{ .PlaygroundVersion }}/build/static/js/middleware.js"></script>
</head>

<body>
  <div id="root">
    <style>
      body {
        background-color: rgb(23, 42, 58);
        font-family: Open Sans, sans-serif;
        height: 90vh;
      }

      #root {
        height: 100%;
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
      }

      .loading {
        font-size: 32px;
        font-weight: 200;
        color: rgba(255, 255, 255, .6);
        margin-left: 20px;
      }

      img {
        width: 78px;
        height: 78px;
      }

      .title {
        font-weight: 400;
      }
    </style>
    <img src='//cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png' alt=''>
    <div class="loading"> Loading
      <span class="title">GraphQL Playground</span>
    </div>
  </div>
  <script>window.addEventListener('load', function (event) {
      GraphQLPlayground.init(document.getElementById('root'), {
        endpoint: {{ .Endpoint }},
        subscriptionEndpoint: {{ .SubscriptionEndpoint }},
        setTitle: {{ .SetTitle }}
      })
    })</script>
</body>

</html>
{{ end }}
`
