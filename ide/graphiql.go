package ide

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/graphql-go/graphql"
)

// GraphiqlVersion is the current version of GraphiQL
var GraphiqlVersion = "1.4.1"

type GraphiQLOptions struct {
	Version              string
	SSL                  bool
	Endpoint             string
	SubscriptionEndpoint string
	Template             string
}

func NewDefaultGraphiQLOptions() *GraphiQLOptions {
	return &GraphiQLOptions{
		Version: GraphiqlVersion,
	}
}

func NewDefaultSSLGraphiQLOption() *GraphiQLOptions {
	return &GraphiQLOptions{
		Version: GraphiqlVersion,
		SSL:     true,
	}
}

// graphiqlData is the page data structure of the rendered GraphiQL page
type graphiqlData struct {
	Endpoint             string
	SubscriptionEndpoint string
	GraphiqlVersion      string
	QueryString          string
	VariablesString      string
	OperationName        string
	ResultString         string
}

// renderGraphiQL renders the GraphiQL GUI
func RenderGraphiQL(config *GraphiQLOptions, w http.ResponseWriter, r *http.Request, params graphql.Params) {
	graphiqlTemplate := GraphiqlTemplateV2
	if config.Template != "" {
		graphiqlTemplate = config.Template
	}

	t := template.New("GraphiQL")
	t, err := t.Parse(graphiqlTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create variables string
	vars, err := json.MarshalIndent(params.VariableValues, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	varsString := string(vars)
	if varsString == "null" {
		varsString = ""
	}

	// Create result string
	var resString string
	if params.RequestString == "" {
		resString = ""
	} else {
		result, err := json.MarshalIndent(graphql.Do(params), "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resString = string(result)
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

	d := graphiqlData{
		GraphiqlVersion:      GraphiqlVersion,
		QueryString:          params.RequestString,
		ResultString:         resString,
		VariablesString:      varsString,
		OperationName:        params.OperationName,
		Endpoint:             endpoint,
		SubscriptionEndpoint: subscriptionEndpoint,
	}
	err = t.ExecuteTemplate(w, "index", d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const GraphiqlTemplateV2 = `
{{ define "index" }}
<!--
 *  Copyright (c) 2021 GraphQL Contributors
 *  All rights reserved.
 *
 *  This source code is licensed under the license found in the
 *  LICENSE file in the root directory of this source tree.
-->
<!doctype html>
<html lang="en">
  <head>
    <title>GraphiQL</title>
    <style>
      body {
        height: 100%;
        margin: 0;
        width: 100%;
        overflow: hidden;
      }

      #graphiql {
        height: 100vh;
      }
    </style>
    <!--
      This GraphiQL example depends on Promise and fetch, which are available in
      modern browsers, but can be "polyfilled" for older browsers.
      GraphiQL itself depends on React DOM.
      If you do not want to rely on a CDN, you can host these files locally or
      include them directly in your favored resource bundler.
    -->
    <script
      crossorigin
      src="https://unpkg.com/react@18/umd/react.production.min.js"
    ></script>
    <script
      crossorigin
      src="https://unpkg.com/react-dom@18/umd/react-dom.production.min.js"
    ></script>
    <!--
      These two files can be found in the npm module, however you may wish to
      copy them directly into your environment, or perhaps include them in your
      favored resource bundler.
     -->
    <script
      src="https://unpkg.com/graphiql/graphiql.min.js"
      type="application/javascript"
    ></script>
    <link rel="stylesheet" href="https://unpkg.com/graphiql/graphiql.min.css" />
    <!-- 
      These are imports for the GraphIQL Explorer plugin.
     -->
    <script
      src="https://unpkg.com/@graphiql/plugin-explorer/dist/index.umd.js"
      crossorigin
    ></script>

    <link
      rel="stylesheet"
      href="https://unpkg.com/@graphiql/plugin-explorer/dist/style.css"
    />
  </head>

  <body>
    <div id="graphiql">Loading...</div>
    <script>
      const subscriptionUrl = window.location.href.replace(/^http/, "ws")
      const root = ReactDOM.createRoot(document.getElementById('graphiql'));
      const fetcher = GraphiQL.createFetcher({
        url: window.location.href,
        subscriptionUrl: subscriptionUrl,
      });
      const explorerPlugin = GraphiQLPluginExplorer.explorerPlugin();
      root.render(
        React.createElement(GraphiQL, {
          fetcher,
          defaultEditorToolsVisibility: true,
          plugins: [explorerPlugin],
        }),
      );
    </script>
  </body>
</html>
{{ end}}
`

const GraphiqlTemplateV1 = `
{{ define "index" }}
<html>
  <head>
    <title>Simple GraphiQL Example</title>
    <link href="https://unpkg.com/graphiql/graphiql.min.css" rel="stylesheet" />
    <script
      crossorigin
      src="https://unpkg.com/react/umd/react.production.min.js"
    ></script>
    <script
      crossorigin
      src="https://unpkg.com/react-dom/umd/react-dom.production.min.js"
    ></script>
    <script
      crossorigin
      src="https://unpkg.com/graphiql/graphiql.js"
    ></script>
  
  </head>
  <body style="margin: 0;">
    <div id="graphiql" style="height: 100vh;"></div>
    <script>
      const subscriptionUrl = window.location.href.replace(/^http/, "ws")
      const fetcher = GraphiQL.createFetcher({
        url: window.location.href,
        subscriptionUrl: subscriptionUrl,
      });

      ReactDOM.render(
        React.createElement(GraphiQL, { fetcher: fetcher }),
        document.getElementById('graphiql'),
      );
    </script>
  </body>
</html>
{{end}}
`
